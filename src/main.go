// Copyright (c) 2021 zacksleo <zacksleo@gmail.com>
// MIT Licence - http://opensource.org/licenses/MIT

/**
* timestamp alfred wordflow
 */
package main

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	aw "github.com/deanishe/awgo"
	"golang.org/x/net/html"
)

var (
	wf          *aw.Workflow
	maxCacheAge = 24 * 90 * time.Hour // How long to cache repo list for
)

func init() {
	wf = aw.New()
}

func help() {
	wf.NewItem("url help").Subtitle("查询帮助")
	wf.NewItem("url {url}").Subtitle("分享当前网址")
	wf.SendFeedback()
}

type HTMLMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	SiteName    string `json:"site_name"`
}

func parse(link string) {
	if _, err := url.Parse(link); err != nil {
		wf.NewItem("error").Subtitle(err.Error())
		wf.SendFeedback()
		return
	}

	meta := new(HTMLMeta)

	cacheKey := getMd5(link) + ".json"
	if wf.Cache.Exists(cacheKey) {
		wf.Cache.LoadJSON(cacheKey, &meta)
	}

	if wf.Cache.Expired(cacheKey, maxCacheAge) {

		resp, err := Get(link)

		if err != nil {
			wf.NewItem("error").Subtitle(err.Error())
			wf.SendFeedback()
			return
		}
		defer resp.Body.Close()
		meta = extract(resp.Body)
	}

	wf.Cache.StoreJSON(cacheKey, meta)
	meta.Title = cleanBreak(pureTitle(meta.Title))
	meta.Description = cleanBreak(meta.Description)
	item := wf.NewItem(fmt.Sprintf("%s [%s]", meta.Title, meta.SiteName)).Subtitle(meta.Description).Valid(true).Var("url", link).Var("title", meta.Title).Var("description", meta.Description).Var("image", meta.Image).Var("siteName", meta.SiteName).Quicklook(link)
	item.Ctrl().Subtitle("复制到 Markdown")
	wf.SendFeedback()
}

func Get(url string) (resp *http.Response, err error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.128 Safari/537.36")

	return client.Do(req)
}

func getMd5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func extract(resp io.Reader) *HTMLMeta {
	z := html.NewTokenizer(resp)

	titleFound := false

	hm := new(HTMLMeta)

	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			return hm
		case html.StartTagToken, html.SelfClosingTagToken:
			t := z.Token()
			if t.Data == `body` {
				return hm
			}
			if t.Data == "title" {
				titleFound = true
			}
			if t.Data == "meta" {
				desc, ok := extractMetaProperty(t, "description")
				if ok {
					hm.Description = desc
				}

				ogTitle, ok := extractMetaProperty(t, "og:title")
				if ok {
					hm.Title = ogTitle
				}

				ogDesc, ok := extractMetaProperty(t, "og:description")
				if ok {
					hm.Description = ogDesc
				}

				ogImage, ok := extractMetaProperty(t, "og:image")
				if ok {
					hm.Image = ogImage
				}

				ogSiteName, ok := extractMetaProperty(t, "og:site_name")
				if ok {
					hm.SiteName = ogSiteName
				}
			}
		case html.TextToken:
			if titleFound {
				t := z.Token()
				hm.Title = t.Data
				titleFound = false
			}
		}
		if len(hm.SiteName) < 1 {
			hm.SiteName = parseSiteNameFromTitle(hm.Title)
		}
	}
	return hm
}

func extractMetaProperty(t html.Token, prop string) (content string, ok bool) {
	for _, attr := range t.Attr {
		if attr.Key == "property" && attr.Val == prop {
			ok = true
		}
		log.Printf("arr=%s, %s", attr.Key, attr.Val)
		if attr.Key == "name" && strings.ToLower(attr.Val) == prop {
			ok = true
		}

		if attr.Key == "content" {
			content = attr.Val
		}
	}
	log.Printf("extractMetaProperty(%s, %s)=%s, %t", t, prop, content, ok)
	return
}

func pureTitle(title string) string {
	str := strings.ReplaceAll(title, "_", "-")
	str = strings.ReplaceAll(str, " ", "")
	ss := strings.Split(str, "-")
	if len(ss) <= 1 {
		return str
	}
	return strings.Join(ss[:len(ss)-1], "")
}

func cleanBreak(description string) string {
	re := regexp.MustCompile(`[\r\n\s]+`)
	str := re.ReplaceAllString(description, " ")
	return str
}

func parseSiteNameFromTitle(title string) string {
	str := strings.ReplaceAll(cleanBreak(title), "_", "-")
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ReplaceAll(str, "－", "-")
	str = strings.ReplaceAll(str, "|", "-")
	ss := strings.Split(str, "-")
	log.Printf("str=%s\n", str)
	return ss[len(ss)-1]
}

func run() {

	query := ""
	if len(wf.Args()) > 0 {
		query = wf.Args()[0]
	}

	if query == `help` {
		help()
		return
	}

	// 默认展示 help
	if len(query) < 1 {
		help()
		return
	}

	re := regexp.MustCompile(`^(((ht|f)tps?):\/\/)?[\w-]+(\.[\w-]+)+([\w.,@?^=%&:/~+#-]*[\w@?^=%&/~+#-])?$`)
	if re.Match([]byte(query)) {
		parse(query)
		return
	}

	wf.NewItem("格式不正确").Subtitle("请重新输入")

	wf.SendFeedback()
}

func main() {
	wf.Run(run)
}
