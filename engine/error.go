package engine

func CheckError(err error) int {
	x := ErrorRecover(err)()
	return x.(int)
}

func ErrorRecover(value interface{}) func() interface{} {
	var x = func() interface{} {
		var z int = 1
		defer func() interface{} {
			if err := recover(); err != nil {
				z = 0
			}
			return z
		}()
		return z
	}
	return x
}
