package main_exit_check_test_data

import (
	"os"
	. "os"
	kek "os"
)

func main() {
	Exit(0)     // want "os.Exit call in main function via dot import"
	kek.Exit(0) // want "os.Exit call in main function via alias"
	os.Exit(0)  // want "os.Exit call in main function"
}
