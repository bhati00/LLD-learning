package main

import "sync"

func main() {
	file, err := NewFile("app.log")
	if err != nil {
		panic(err)
	}
	log := New([]Destination{NewConsole(), file}, WithMinLevel(Debug))
	defer log.Close()

	log.Debug("debugging details")
	log.Info("server started")
	log.Warn("disk almost full")
	log.Error("db connection failed")
	log.Fatal("cannot continue")

	// Many goroutines writing at once: every line must come out whole.
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Info("hello from a goroutine")
		}()
	}
	wg.Wait()
}
