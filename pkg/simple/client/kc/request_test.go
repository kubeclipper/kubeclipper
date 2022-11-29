package kc

import (
	"net/url"
	"testing"

	"github.com/kubeclipper/kubeclipper/pkg/query"
)

func TestQueries_ToRawQuery(t *testing.T) {
	q := query.New()
	q.LabelSelector = "a=b"
	q.FieldSelector = "metadata.name=demo"
	q.Watch = true
	q.FuzzySearch = map[string]string{
		"name": "demo",
	}
	kcq := Queries(*q)
	rawQuery := kcq.ToRawQuery()
	got := rawQuery.Encode()

	target := make(url.Values)
	target.Set("paging", "limit=-1,page=0")
	target.Set("watch", "true")
	target.Set("fieldSelector", "metadata.name=demo")
	target.Set("labelSelector", "a=b")
	target.Set("fuzzy", "name~demo")
	want := target.Encode()

	if got != want {
		t.Fatalf("want:%s got:%s\n", want, got)
	}
}
