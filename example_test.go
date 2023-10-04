package rlzone_test

import (
	"fmt"
	"log"
	"time"

	"github.com/Snawoot/rlzone"
)

func ExampleSimple() {
	rl, err := rlzone.NewSmallest[string](1*time.Minute, 5)
	if err != nil {
		log.Fatalf("unable to create ratelimit instance: %v", err)
	}
	for i := 0; i < 6; i++ {
		fmt.Println(rl.Allow("user1"))
	}
	fmt.Println(rl.Allow("user2"))
	// Output:
	// true
	// true
	// true
	// true
	// true
	// false
	// true
}
