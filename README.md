# rlzone

[![go-doc](https://godoc.org/github.com/Snawoot/rlzone?status.svg)](https://godoc.org/github.com/Snawoot/rlzone)

Generic rate limit by key using sliding window algorithm. See Cloudflare blog for details: https://blog.cloudflare.com/counting-things-a-lot-of-different-things/

## Example

```golang
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Snawoot/rlzone"
)

func main() {
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
```
