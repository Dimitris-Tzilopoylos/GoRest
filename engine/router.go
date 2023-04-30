package engine

import (
	database "application/database"
	"application/environment"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Purple = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

type HttpVerb string
type RequestContextKey string
type HttpVerbs []HttpVerb
type HttpParam struct {
	paramType string
	key       string
	index     int
}

const (
	GET    HttpVerb = "GET"
	POST   HttpVerb = "POST"
	PUT    HttpVerb = "PUT"
	PATCH  HttpVerb = "PATCH"
	DELETE HttpVerb = "DELETE"
)

var HttpVerbsSlice HttpVerbs = HttpVerbs{GET, POST, PUT, PATCH, DELETE}

type Route struct {
	path        string
	regexPath   string
	handler     http.HandlerFunc
	params      map[string]HttpParam
	preHandlers []func(res http.ResponseWriter, req *http.Request, next func(req *http.Request))
}

type RouterMwConfig struct {
	handler  func(res http.ResponseWriter, req *http.Request, next func(req *http.Request))
	priority int64
}

type Router struct {
	entryUrl          string
	routes            map[HttpVerb]map[string]*Route
	urlKeys           map[HttpVerb][]string
	mwPreHandlers     map[string][]RouterMwConfig
	mwPriority        int64
	Engine            *database.Engine
	Logger            *zap.Logger
	EnableHttpLogging bool
	EnableSQLLogging  bool
}

func (r *Router) Status(res http.ResponseWriter, statusCode int) *Router {

	if statusCode != 200 {
		res.WriteHeader(statusCode)
	}
	return r
}

func (r *Router) Json(res http.ResponseWriter, statusCode int, value interface{}) int {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Engine-Status-Code", fmt.Sprintf("%d", statusCode))
	res.WriteHeader(statusCode)
	switch val := value.(type) {
	case []byte:
		res.Write(val)
	case map[string][]byte:
		err := json.NewEncoder(res).Encode(val)
		if err != nil {
			panic(err.Error())
		}
		// res.Write(data)
	default:
		err := json.NewEncoder(res).Encode(value)
		if err != nil {
			err := json.NewEncoder(res).Encode([]byte{})
			if err != nil {
				panic(err.Error())
			}
		}
	}
	return statusCode
}

func (r *Router) ErrorResponse(res http.ResponseWriter, status int, errorText string) {
	r.Json(res, status, map[string]string{"message": errorText})
}

func (r *Router) NotFound(res http.ResponseWriter, req *http.Request) {
	r.Json(res, 404, map[string]string{"message": "NOT_FOUND"})
}

func NewApp(db *sql.DB) *Router {
	r := &Router{}
	r.Initialize()
	r.Engine = database.Init(db)
	return r
}

func (r *Router) Initialize() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	r.EnableHttpLogging = environment.GetEnvValue("HTTP_LOGGER") == "ON"
	r.EnableSQLLogging = environment.GetEnvValue("SQL_LOGGER") == "ON"
	r.Logger = logger
	r.entryUrl = environment.GetEnvValue("ROUTER_ENTRY_POINT")
	r.routes = make(map[HttpVerb]map[string]*Route)
	r.urlKeys = make(map[HttpVerb][]string)
	r.mwPreHandlers = make(map[string][]RouterMwConfig)
	r.mwPriority = 0
	for _, httpVerb := range HttpVerbsSlice {
		r.urlKeys[httpVerb] = []string{}
		r.routes[httpVerb] = make(map[string]*Route)
	}
}

func (r *Router) GetApplicationContext() *Router {
	return r
}

func (r *Router) MatchRoute(url string, method HttpVerb, urlKeys []string) (string, map[string]string) {
	for _, urlKey := range urlKeys {
		if !strings.Contains(urlKey, ":") {
			continue
		}
		route := r.routes[method][urlKey]
		regex := route.regexPath
		matchRegxp := regexp.MustCompile(regex)
		if !matchRegxp.MatchString(url) {
			continue
		}
		urlPlaceHolders := strings.Split(url, "/")
		params := map[string]string{}
		for key, val := range route.params {
			params[key] = urlPlaceHolders[val.index]
		}
		return urlKey, params
	}
	return "", nil
}

func buildRegexUrlPath(url string) (string, map[string]HttpParam) {
	searchRegexStr := "<[a-zA-Z0-9_-]+:[a-zA-Z0-9_-]+>"
	extractMatchRegex := regexp.MustCompile(searchRegexStr)
	all := extractMatchRegex.FindAllString(url, -1)
	if all == nil {
		url := "^" + url + `\/?$`
		return url, map[string]HttpParam{}
	}
	regex := "^"
	pathRegex := extractMatchRegex.ReplaceAll([]byte(url), []byte(`[a-zA-Z0-9_-]+`))
	regex += string(pathRegex)
	regex += `\/?$`
	params := map[string]HttpParam{}
	for index, val := range strings.Split(url, "/") {
		if len(val) > 0 && extractMatchRegex.MatchString(val) {
			makeParams := strings.Split(string(val), ":")
			makeKey := makeParams[1][:len(makeParams[1])-1]
			makeType := makeParams[0][1:]
			params[makeKey] = HttpParam{paramType: makeType, key: makeKey, index: index}
		}
	}
	return regex, params
}

func (r *Router) GetUrlWithEntryRoute(url string) string {
	if len(r.entryUrl) > 0 {
		return fmt.Sprintf("%s%s", r.entryUrl, url)
	}

	return url
}

func (r *Router) Use(url string, handler func(res http.ResponseWriter, req *http.Request, next func(req *http.Request))) {
	url = r.GetUrlWithEntryRoute(url)
	if _, ok := r.mwPreHandlers[url]; !ok {
		r.mwPreHandlers[url] = make([]RouterMwConfig, 0)
	}
	r.mwPreHandlers[url] = append(r.mwPreHandlers[url], RouterMwConfig{handler: handler, priority: r.mwPriority})
	r.mwPriority += 1

	fmt.Printf("Registering Middleware On %s\n", url)
}

func (r *Router) deriveMiddlewarePreHandlersForRoute(url string) []func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
	handlers := []RouterMwConfig{}
	if preHandlers, ok := r.mwPreHandlers[url]; ok {
		handlers = append(handlers, preHandlers...)
	}
	if wildCardPreHandlers, ok := r.mwPreHandlers["*"]; ok {
		handlers = append(handlers, wildCardPreHandlers...)
	}
	sort.Slice(handlers, func(i, j int) bool {
		return handlers[i].priority < handlers[j].priority
	})
	preHandlers := []func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)){}
	for _, config := range handlers {
		preHandlers = append(preHandlers, config.handler)
	}
	return preHandlers
}

func (r *Router) Get(url string, handler http.HandlerFunc) {
	url = r.GetUrlWithEntryRoute(url)
	regexPath, params := buildRegexUrlPath(url)
	fmt.Printf("Registering Route: [%v]  %s\n", GET, url)
	(*r).routes[GET][url] = &Route{
		path:        url,
		regexPath:   regexPath,
		handler:     r.EngineHandlerWithContext(handler),
		params:      params,
		preHandlers: r.deriveMiddlewarePreHandlersForRoute(url),
	}
}

func (r *Router) Post(url string, handler http.HandlerFunc) {
	url = r.GetUrlWithEntryRoute(url)
	regexPath, params := buildRegexUrlPath(url)
	fmt.Printf("Registering Route: [%v]  %s\n", POST, url)
	(*r).routes[POST][url] = &Route{
		path:        url,
		regexPath:   regexPath,
		handler:     r.EngineHandlerWithContext(handler),
		params:      params,
		preHandlers: r.deriveMiddlewarePreHandlersForRoute(url),
	}
}

func (r *Router) Put(url string, handler http.HandlerFunc) {
	url = r.GetUrlWithEntryRoute(url)
	regexPath, params := buildRegexUrlPath(url)
	fmt.Printf("Registering Route: [%v]  %s\n", PUT, url)
	(*r).routes[PUT][url] = &Route{
		path:        url,
		regexPath:   regexPath,
		handler:     r.EngineHandlerWithContext(handler),
		params:      params,
		preHandlers: r.deriveMiddlewarePreHandlersForRoute(url),
	}
}

func (r *Router) Patch(url string, handler http.HandlerFunc) {
	url = r.GetUrlWithEntryRoute(url)
	regexPath, params := buildRegexUrlPath(url)
	fmt.Printf("Registering Route: [%v]  %s\n", PATCH, url)
	(*r).routes[PATCH][url] = &Route{
		path:        url,
		regexPath:   regexPath,
		handler:     r.EngineHandlerWithContext(handler),
		params:      params,
		preHandlers: r.deriveMiddlewarePreHandlersForRoute(url),
	}
}

func (r *Router) Delete(url string, handler http.HandlerFunc) {
	url = r.GetUrlWithEntryRoute(url)
	regexPath, params := buildRegexUrlPath(url)
	fmt.Printf("Registering Route: [%v]  %s\n", DELETE, url)
	(*r).routes[DELETE][url] = &Route{
		path:        url,
		regexPath:   regexPath,
		handler:     r.EngineHandlerWithContext(handler),
		params:      params,
		preHandlers: r.deriveMiddlewarePreHandlersForRoute(url),
	}
}

func GetHttpMethod(req *http.Request) HttpVerb {
	return HttpVerb(req.Method)
}

func GetUrlHost(req *http.Request) string {
	return req.URL.Host
}

func GetRequestIP(r *http.Request) string {
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if netIP != nil {
		return ip
	}

	//Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if netIP != nil {
			return ip
		}
	}

	//Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	netIP = net.ParseIP(ip)
	if netIP != nil {
		return ip
	}
	return ""
}

func GetUrlPath(req *http.Request) string {
	return req.URL.Path
}

func GetAuth(req *http.Request) jwt.MapClaims {
	return req.Context().Value("auth").(jwt.MapClaims)
}

func GetParams(req *http.Request) map[string]string {
	params := req.Context().Value(RequestContextKey("params"))
	if params == nil {
		return map[string]string{}
	}
	return params.(map[string]string)
}

func GetBody(req *http.Request) map[string]interface{} {
	body, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	if err != nil {
		CheckError(err)
		return make(map[string]interface{})
	}
	var msg map[string]interface{}
	err = json.Unmarshal(body, &msg)
	if err != nil {
		CheckError(err)
		return make(map[string]interface{})
	}
	return msg
}

func EngageBodyToStruct(req *http.Request, object any) (any, error) {
	defer req.Body.Close()

	err := json.NewDecoder(req.Body).Decode(object)
	if err != nil {
		return nil, err
	}
	return object, nil
}

func SetContextValue[T any](req *http.Request, key string, value T) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), RequestContextKey(key), value))
}

func GetContextValue(req *http.Request, key string) any {
	return req.Context().Value(RequestContextKey(key))
}

func GetEngineState(req *http.Request) *Router {
	appContext := req.Context().Value(RequestContextKey("engineState"))
	router := appContext.(*Router)
	return router
}

func HandlerWithContext(handler http.HandlerFunc, key string, ctx interface{}) http.HandlerFunc {
	reqContextKey := RequestContextKey(key)
	return func(res http.ResponseWriter, req *http.Request) {
		req = req.WithContext(context.WithValue(req.Context(), reqContextKey, ctx))
		handler(res, req)
	}
}

func MiddleWareHandlerWithContext(handler func(res http.ResponseWriter, req *http.Request, next func(*http.Request)), key string, ctx interface{}) func(res http.ResponseWriter, req *http.Request, next func(*http.Request)) {
	reqContextKey := RequestContextKey(key)
	return func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)) {
		req = req.WithContext(context.WithValue(req.Context(), reqContextKey, ctx))
		handler(res, req, next)
	}
}

func (r *Router) GetHttpHandler(url string, method HttpVerb) (http.HandlerFunc, []func(res http.ResponseWriter, req *http.Request, next func(req *http.Request))) {
	route := r.routes[method][url]
	if route != nil {
		handler := route.handler
		prehandlers := route.preHandlers
		if handler != nil {
			return handler, prehandlers
		}
	}

	matchedUrl, params := r.MatchRoute(url, method, r.urlKeys[method])
	route = r.routes[method][matchedUrl]
	if route == nil {
		return r.NotFound, []func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)){}
	}
	handler := route.handler
	prehandlers := route.preHandlers

	if handler != nil {
		preHandlersWithContext := []func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)){}
		for _, preHandler := range prehandlers {
			preHandlersWithContext = append(preHandlersWithContext, MiddleWareHandlerWithContext(preHandler, "params", params))
		}
		return HandlerWithContext(handler, "params", params), preHandlersWithContext
	}
	return r.NotFound, []func(res http.ResponseWriter, req *http.Request, next func(req *http.Request)){}
}

func (r *Router) LogRequest(w *http.Request, res http.ResponseWriter) {
	dt := time.Now()
	currentRequestTime := dt.Local()
	formattedDate := currentRequestTime.Format("January 02, 2006 15:04:05")
	method := GetHttpMethod(w)
	path := GetUrlPath(w)
	ip := GetRequestIP(w)
	statusCode := res.Header().Get("Engine-Status-Code")

	statusCheck, err := strconv.Atoi(statusCode)
	var statusText string

	if err == nil {
		statusText = http.StatusText(statusCheck)
		if statusCheck >= 400 {
			r.Logger.Error("HTTP Request",
				zap.String("timestamp", formattedDate),
				zap.Any("method", method),
				zap.String("path", path),
				zap.String("statusCode", statusCode+" "+statusText),
				zap.String("ip", ip),
				zap.Any("requestId", w.Context().Value(RequestContextKey("requestId"))))
		} else {
			r.Logger.Info("HTTP Request",
				zap.String("timestamp", formattedDate),
				zap.Any("method", method),
				zap.String("path", path),
				zap.String("statusCode", statusCode+" "+statusText),
				zap.String("ip", ip),
				zap.Any("requestId", w.Context().Value(RequestContextKey("requestId"))))
		}
	} else {
		ErrorRecover(err)()
	}

}

func (r *Router) withRequestId(req *http.Request) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), RequestContextKey("requestId"), uuid.New().String()))
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "*")
}

func next(preHandlers []func(res http.ResponseWriter, req *http.Request, next func(*http.Request)), handler http.HandlerFunc, res http.ResponseWriter, req *http.Request, i int) {
	preHandlers[i](res, req, func(req *http.Request) {
		if i+1 < len(preHandlers) {
			next(preHandlers, handler, res, req, i+1)
		} else {
			handler(res, req)
		}
	})
}

func (r *Router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	enableCors(&res)
	httpMethod := GetHttpMethod(req)
	if httpMethod == "OPTIONS" {
		r.Json(res, 204, "")
		return
	}
	req = req.WithContext(context.Background())
	url := GetUrlPath(req)
	handler, preHandlers := r.GetHttpHandler(url, httpMethod)
	req = r.withRequestId(req)
	if handler != nil {
		if len(preHandlers) > 0 {
			next(preHandlers, handler, res, req, 0)
		} else {
			handler(res, req)
		}
		if r.EnableHttpLogging {
			r.LogRequest(req, res)
		}

		return
	}
	panic("NO HTTP HANDLER FOUND FOR REQUEST")
}

func (r *Router) populateUrlKeys() {
	for key := range r.routes {
		for urlKey := range r.routes[key] {
			r.urlKeys[key] = append(r.urlKeys[key], urlKey)
		}
	}
}

func (r *Router) Listen() {
	port := environment.GetEnvValue("PORT")
	r.populateUrlKeys()
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	version := environment.GetEnvValue("VERSION")
	fmt.Printf("GOJila Engine Version: %s\n", version)
	fmt.Printf("Server started at: http://localhost%s\n", port)
	err := server.ListenAndServe()
	if err != nil {
		panic(err.Error())
	}

}

func (r *Router) EngineHandlerWithContext(handler http.HandlerFunc) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		req = req.WithContext(context.WithValue(req.Context(), RequestContextKey("engineState"), r))
		handler(res, req)
	}
}

func (r *Router) Subscribe(eventName string, listener func(args ...any)) func() {
	return r.Engine.EventEmitter.Subscribe(eventName, listener)
}

func (r *Router) RemoveAllLEventEmitteristeners() {
	for key := range r.Engine.EventEmitter.EventMap {
		delete(r.Engine.EventEmitter.EventMap, key)
	}
}

func (r *Router) EmitEvent(eventName string, args ...any) {
	r.Engine.EventEmitter.Emit(eventName, args...)
}
