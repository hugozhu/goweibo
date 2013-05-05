goweibo
=======

Weibo SDK for Golang

# Usage

Save weibo OAuth2 token to a local file named 'token', eg. "2.008TkTLDIQdqsD4bbfd082cchG3ABC"

```
package main

import (
    "github.com/hugozhu/goweibo"
    "log"
)

var sina = &weibo.Sina{
    AccessToken: weibo.ReadToken("./token"),
}

func main() {
    //fetch 20 weibo after 12345678
    for _, p := range sina.TimeLine(0, "hugozhu", 12345678, 20) {
        log.Println(p)
    }
}
```

# Author

Hugo Zhu (http://hugozhu.myalert.info)