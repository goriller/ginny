package mongodb

import (
	"context"
	"io/ioutil"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/yaml.v2"
)

type book struct {
	Title  string
	Author string
	Tags   primitive.A
}

// 连接测试：无认证模式、用户名密码模式。断点查看：clientOptions赋值
func TestNewManager(t *testing.T) {
	data, err := ioutil.ReadFile("./mongodb_config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		t.Fatal(err)
	}
	t.Log(cfg)

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}

	test1Coll := mgr.DB().Collection("test1")
	doc := &book{
		Title:  "The Polyglot Developer Podcast",
		Author: "Nic Raboy",
		Tags:   bson.A{"development", "programming", "coding"},
	}
	insertRes, err := test1Coll.InsertOne(context.Background(), &doc)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(insertRes.InsertedID)
}
