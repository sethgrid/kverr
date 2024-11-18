# KVErr

Package `kverr` provides for key:value pair error values that can be bubbled up from lower level function calls to allow structured key:value paired logging at higher levels.

Any number of nested calls may be wrapped with an additional `kverr.New()`, and any additional key:value pairs will be added onto any existing error context and bubbled up.

```
func skipLevel() error {
	if err := returnKVErr(); err != nil {
		return fmt.Errorf("oh noes, err: %w", err)
	}
	return nil
}

func returnKVErr() error {
	return New(fmt.Errorf("root error"), "kv_present", true)
}

func doThingWithLogs() error {
    err := skipLevel()

    // args will be a list of kv pairs ["kv_present", true]
    args := YoinkArgs(err) 
    logger.Error("had an error", args...)
}


```