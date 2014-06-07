package weibo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hugozhu/log4go"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const BaseURL = "https://api.weibo.com/2"

var log = log4go.New(os.Stdout)

func SetDebugEnabled(enable *bool) {
	log.DebugEnabled = enable
}

type Sina struct {
	AccessToken string
}

type UserKeyword struct {
	Keyword       string
	WeiboUid      int64
	WeiboUsername string
	Id            int64
	Frequence     int64
}

type Weibo struct {
	Id        int64
	WeiboId   int64
	Status    int
	LastId    int64
	WeiboName string
	Created   int64
	Modified  int64
}

type WeiboPosts struct {
	Statuses []*WeiboPost
}

type WeiboPost struct {
	Created_At              string
	Id                      int64
	Mid                     string
	Text                    string
	Source                  string
	Trucated                bool
	In_Reply_To_Status_Id   string
	In_Reply_To_Screen_Name string
	Thumbnail_Pic           string
	Bmiddle_Pic             string
	Original_Pic            string
	User                    *WeiboUser
	Retweeted_Status        *WeiboPost
	Reposts_Count           int
	Comments_Count          int
	Attitudes_Count         int
	Link                    string
}

type WeiboUser struct {
	Id                int64
	Screen_name       string
	Name              string
	Location          string
	Description       string
	Url               string
	Profile_Image_Url string
	Verified_Reason   string
}

type WeiboMention struct {
	Statuses        []WeiboPost
	Hasvisible      bool
	Previous_cursor int
	Next_cursor     int
	Total_number    int
	Interval        int
}

type WeiboComment struct {
	Id     int64
	Text   string
	Source string
	Mid    string
	User   *WeiboUser
	Status *WeiboPost
}

type WeiboUrlInfos struct {
	Urls []*WeiboUrlInfo
}

type WeiboUrlInfo struct {
	Url_Short   string
	Url_Long    string
	Title       string
	Description string
}

type WeiboMid struct {
	Mid string
}

type WeiboError struct {
	Err        string `json:"Error"`
	Error_Code int64
	Request    string
}

func (e WeiboError) Error() string {
	return fmt.Sprintf("%d %s %s", e.Error_Code, e.Err, e.Request)
}

func (s *Sina) TimeLine(uid int64, screen_name string, since_id int64, count int) []*WeiboPost {
	params := url.Values{}
	if uid > 0 {
		params.Set("uid", strconv.FormatInt(uid, 10))
	} else if screen_name != "" {
		params.Set("screen_name", screen_name)
	}

	params.Set("since_id", strconv.FormatInt(since_id, 10))
	params.Set("count", strconv.Itoa(count))
	var posts WeiboPosts
	if s.GET("/statuses/user_timeline.json", params, &posts) {
		return posts.Statuses
	}
	return nil
}

func (s *Sina) UsersShow(uid int64) *WeiboUser {
	params := url.Values{}
	params.Set("uid", strconv.FormatInt(uid, 10))
	var v WeiboUser
	if s.GET("/users/show.json", params, &v) {
		return &v
	}
	return nil
}

func (s *Sina) CommentsCreate(id int64, comment string) *WeiboComment {
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("comment", comment)
	var v WeiboComment
	if s.POST("/comments/create.json", params, &v) {
		return &v
	}
	return nil
}

func (s *Sina) StatusesShow(postId int64) (v *WeiboPost) {
	params := url.Values{}
	params.Set("id", strconv.FormatInt(postId, 10))
	s.GET("/statuses/show.json", params, &v)
	return
}

func (s *Sina) StatusesRepost(id int64, status string) *WeiboPost {
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("status", status)
	params.Set("is_comment", "0")
	var v WeiboPost
	if s.POST("/statuses/repost.json", params, &v) {
		return &v
	}
	return nil
}

func (s *Sina) StatusesUpload(status string, reader io.Reader) *WeiboPost {
	params := url.Values{}
	params.Set("status", status)
	var v WeiboPost
	if ok, _ := s.UPLOAD("/statuses/upload.json", params, "pic", "filename", reader, &v); ok {
		return &v
	}
	return nil
}

func (s *Sina) Mentions() (mentions *WeiboMention) {
	params := url.Values{}
	s.GET("/statuses/mentions.json", params, &mentions)
	return mentions
}

func (s *Sina) ShortUrlInfo(urls []string) []*WeiboUrlInfo {
	params := url.Values{}
	for _, u := range urls {
		params.Add("url_short", u)
	}
	var v WeiboUrlInfos
	if s.GET("/short_url/info.json", params, &v) {
		return v.Urls
	}
	return nil
}

func (s *Sina) QueryMid(id int64, typ int) string {
	params := url.Values{}
	params.Set("id", strconv.FormatInt(id, 10))
	params.Set("type", fmt.Sprintf("%d", typ))
	var v *WeiboMid
	s.GET("/statuses/querymid.json", params, &v)
	return v.Mid
}

func ExpandUrls(urls []string) (expandedUrls []string) {
	for _, u := range urls {
		client := &http.Client{
			CheckRedirect: func(req *http.Request, reqs []*http.Request) error {
				return errors.New("No need to redirect")
			},
		}
		resp, _ := client.Get(u)
		if resp != nil {
			defer resp.Body.Close()
			if resp.Header["Location"] != nil {
				expandedUrls = append(expandedUrls, resp.Header.Get("Location"))
			}
		}
	}
	return
}

func (s *Sina) weiboApi(method string, base string, query url.Values, v interface{}) (bool, error) {
	url1 := BaseURL + base
	query.Set("access_token", s.AccessToken)

	var resp *http.Response
	var err error
	if method == "POST" {
		resp, err = http.PostForm(url1, query)
	} else {
		url1 += "?" + query.Encode()
		resp, err = http.Get(url1)
	}
	if err != nil {
		log.Errorf("fetch url %s %s", url1, err)
		panic(err)
	}
	defer resp.Body.Close()

	log.Debug(url1)

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode == 200 {
		err = d.Decode(&v)
		if err != nil {
			panic(err)
		}
		return true, err
	} else {
		bytes, _ := ioutil.ReadAll(resp.Body)
		log.Error("Weibo API Error: " + string(bytes))
		var e WeiboError
		err = json.Unmarshal(bytes, &e)
		if err != nil {
			panic(err)
		}
		return false, e
	}
	return false, err
}

func http_call(req *http.Request, v interface{}) (bool, error) {
	if req.Method == "POST" {
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("fetch url %s %s", req.RequestURI, err)
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		d := json.NewDecoder(resp.Body)
		err = d.Decode(&v)
		if err != nil {
			panic(err)
		}
		return true, err
	} else {
		bytes, _ := ioutil.ReadAll(resp.Body)
		log.Error("Weibo API Error: " + string(bytes))
		var e WeiboError
		err = json.Unmarshal(bytes, &e)
		if err != nil {
			panic(err)
		}
		return false, e
	}
	return false, err
}

func (s *Sina) GET(base string, query url.Values, v interface{}) bool {
	ok, err := s.weiboApi("GET", base, query, v)
	if !ok && err != nil {
		log.Error("Weibo GET API Error:", base, query)
		code := err.(WeiboError).Error_Code
		if code != 20101 {
			msg := err.(WeiboError).Err + " " + err.(WeiboError).Request
			ReportFatalError(msg)
		}
	}
	return ok
}

func (s *Sina) POST(base string, query url.Values, v interface{}) bool {
	ok, err := s.weiboApi("POST", base, query, v)
	if !ok && err != nil {
		// code := err.(WeiboError).Error_Code
		log.Error("Weibo POST API Error:", base, query)
		msg := err.(WeiboError).Err + " " + err.(WeiboError).Request
		ReportFatalError(msg)
	}
	return ok
}

func (s *Sina) UPLOAD(base string, params url.Values,
	uploadFieldName string,
	file string,
	reader io.Reader, v interface{}) (bool, error) {
	params.Set("access_token", s.AccessToken)
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	for k, _ := range params {
		w.WriteField(k, params.Get(k))
	}
	wr, _ := w.CreateFormFile(uploadFieldName, filepath.Base(file))
	if reader != nil {
		io.Copy(wr, reader)
	}
	w.Close()
	req, err := http.NewRequest("POST", BaseURL+base, buf)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return http_call(req, v)
}

func ReadToken(token string) string {
	data, err := ioutil.ReadFile(os.Getenv("PWD") + "/" + token)
	if err != nil {
		log.Error(err)
		panic(err)
	}
	return string(data[:32])
}

func ReadLastId(file string) (id int64) {
	data, err := ioutil.ReadFile(os.Getenv("PWD") + "/" + file)
	if err != nil {
		log.Error(err)
		panic(err)
	}
	id, err = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		log.Error(err)
		panic(err)
	}
	return
}

func WriteLastId(file string, id int64) {
	err := ioutil.WriteFile(os.Getenv("PWD")+"/"+file, []byte(strconv.FormatInt(id, 10)), 0700)
	if err != nil {
		log.Error(err)
		panic(err)
	}
}

var one_shot_alert = &sync.Once{}

func ReportFatalError(msg string) {
	one_shot_alert.Do(func() {
		log.Error("Fatal Error:", msg)
		os.Exit(-1)
	})
}
