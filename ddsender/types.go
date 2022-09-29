package ddsender

import (
	"sync"
	"time"
)

type MessageType string

const (
	Text       MessageType = "text"
	Link       MessageType = "link"
	Markdown   MessageType = "markdown"
	ActionCard MessageType = "actionCard"
	FeedCard   MessageType = "feedCard"
)

type Message struct {
	MsgType    MessageType    `json:"msgtype"`
	Markdown   MarkdownType   `json:"markdown,omitempty"`
	Text       TextType       `json:"text,omitempty"`
	At         AtType         `json:"at,omitempty"`
	ActionCard ActionCardType `json:"actionCard,omitempty"`
	Link       LinkType       `json:"link,omitempty"`
	FeedCard   FeedCardType   `json:"feedCard,omitempty"`
}

type AtType struct {
	AtMobiles []string `json:"atMobiles"`
	AtUserIds []string `json:"atUserIds"`
	IsAtAll   bool     `json:"isAtAll"`
}

type MarkdownType struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type TextType struct {
	Content string `json:"content"`
}

type ActionCardType struct {
	Title          string `json:"title"`
	Text           string `json:"text"`
	SingleTitle    string `json:"singleTitle"`
	SingleURL      string `json:"singleURL"`
	BtnOrientation string `json:"btnOrientation"`
	Btns           `json:"btns"`
	HideAvatar     string `json:"hideAvatar"`
}

type Btns struct {
	Title     string `json:"title"`
	ActionURL string `json:"actionURL"`
}

type LinkType struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	PicURL     string `json:"picUrl"`
	MessageURL string `json:"messageUrl"`
}

type FeedCardType struct {
	Links []LinkType `json:"links"`
}

type Response struct {
	Errcode int32  `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

type IgnoreMessageType struct {
	MessageHash []string
	Time        string
}

type limitAttribeType struct {
	total  int
	expire time.Time
}

type cacheSenderCountType struct {
	data map[string]*limitAttribeType
	lock *sync.RWMutex
}

func (c *cacheSenderCountType) GetCount(key string) (*limitAttribeType, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	item, ok := c.data[key]
	return item, ok
}

func (c *cacheSenderCountType) Add(key string) int {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.data[key].total++
	return c.data[key].total
}

func (c *cacheSenderCountType) Reset(key string, expire time.Time) *limitAttribeType {
	c.lock.Lock()
	defer c.lock.Unlock()
	new_data := &limitAttribeType{
		total:  0,
		expire: expire,
	}
	c.data[key] = new_data
	return new_data
}

func (c *cacheSenderCountType) Del(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.data, key)
}
