package database

import (
	"application/environment"
	"fmt"
)

func GetGraphiqlTemplate() string {
	var baseUrl string = environment.GetEnvValue("ROUTER_ENTRY_POINT")
	var graphqlEndpoint string = environment.GetEnvValueToStringWithDefault("GRAPHIQL_ENDPOINT", "/graphql")
	return fmt.Sprintf(`<!DOCTYPE html>
	<html lang="en">
	<head>
		<title>GraphiQL</title>
		<style>
		body {
			height: 100%%;
			margin: 0;
			width: 100%%;
			overflow: hidden;
		}

		#graphiql {
			height: 100vh;
		}
		</style>

		<!--
		This GraphiQL example depends on Promise and fetch, which are available in
		modern browsers, but can be "polyfilled" for older browsers.
		GraphiQL itself depends on React DOM.
		If you do not want to rely on a CDN, you can host these files locally or
		include them directly in your favored resource bundler.
		-->
		<script
		src="https://unpkg.com/react@17/umd/react.development.js"
		integrity="sha512-Vf2xGDzpqUOEIKO+X2rgTLWPY+65++WPwCHkX2nFMu9IcstumPsf/uKKRd5prX3wOu8Q0GBylRpsDB26R6ExOg=="
		crossorigin="anonymous"
		></script>
		<script
		src="https://unpkg.com/react-dom@17/umd/react-dom.development.js"
		integrity="sha512-Wr9OKCTtq1anK0hq5bY3X/AvDI5EflDSAh0mE9gma+4hl+kXdTJPKZ3TwLMBcrgUeoY0s3dq9JjhCQc7vddtFg=="
		crossorigin="anonymous"
		></script>

		<!--
		These two files can be found in the npm module, however you may wish to
		copy them directly into your environment, or perhaps include them in your
		favored resource bundler.
		-->
		<link rel="stylesheet" href="https://unpkg.com/graphiql/graphiql.min.css" />
	</head>

	<body>
		<div id="graphiql">Loading...</div>
		<script
		src="https://unpkg.com/graphiql/graphiql.min.js"
		type="application/javascript"
		></script>
		<script>
		const urlParams = new URLSearchParams(window.location.search);
		const token = urlParams.get('token');
		 
		ReactDOM.render(
			React.createElement(GraphiQL, {
			fetcher: GraphiQL.createFetcher({
				url: '%s%s',
				headers: {Authorization: "Bearer "+token}
			}),
			defaultEditorToolsVisibility: true,
			}),
			document.getElementById('graphiql'),
		);
		</script>
	</body>
	</html>
		`,
		baseUrl,
		graphqlEndpoint,
	)
}
