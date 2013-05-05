goweibo
=======

Weibo SDK for Golang

# Usage

Save weibo OAuth2 token to a local file

```
var sina = &weibo.Sina{
    AccessToken: weibo.ReadToken("./token"),
}

func main() {
    for _, p := range sina.TimeLine(0, "hugozhu", 0, 20) {
        log.Println(p)
    }
}
```