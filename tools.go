package webizen

import (
	// "log"
	"strings"

	"github.com/kierdavis/argo"
	"github.com/linkeddata/gold"
)

var (
	foaf = argo.NewNamespace("http://xmlns.com/foaf/0.1/")
)

func assertURI(uri string) (uris map[string]int64) {
	uris = make(map[string]int64)
	names := make(map[string]string)
	mboxes := make(map[string]string)
	images := make(map[string]string)

	g := gold.NewGraph(uri)
	err := g.LoadURI(uri)
	if err != nil {
		return
	}

	for elt := range g.IterTriples() {
		var k, v string

		switch s := elt.Subject.(type) {
		case *argo.Resource:
			k = s.URI
		}
		switch o := elt.Object.(type) {
		case *argo.Resource:
			v = o.URI
		case *argo.Literal:
			v = o.Value
		}

		if elt.Predicate.Equal(foaf.Get("name")) {
			uris[k] = 0
			names[k] = v
		}
		if elt.Predicate.Equal(foaf.Get("img")) ||
			elt.Predicate.Equal(foaf.Get("depiction")) {
			uris[k] = 0
			images[k] = v
		}
		if elt.Predicate.Equal(foaf.Get("mbox")) {
			uris[k] = 0
			mboxes[k] = v
		}
	}

	for k, _ := range uris {
		user := &User{Uri: k}
		db.InsertOne(user)
		db.Get(user)
		uris[k] = user.Id
	}
	for k, v := range names {
		db.Delete(&UserName{User: uris[k]})
		db.InsertOne(&UserName{uris[k], v})
	}
	for k, v := range images {
		db.Delete(&UserImage{User: uris[k]})
		db.InsertOne(&UserImage{uris[k], v})
	}
	for k, v := range mboxes {
		db.Delete(&UserMbox{User: uris[k]})
		db.InsertOne(&UserMbox{uris[k], v})
	}

	return uris
}

type result struct {
	Image []string `json:"image,omitempty"`
	Mbox  []string `json:"mbox,omitempty"`
	Name  []string `json:"name,omitempty"`
}

func search(query string) (r map[string]result) {
	r = map[string]result{}
	cache := map[int64]string{}

	for _, elt := range strings.Split(query, " ") {
		if len(elt) < 6 {
			continue
		}
		if elt[:6] == "https:" || elt[:5] == "http:" {
			for k, v := range assertURI(elt) {
				cache[v] = k
			}
		}
	}

	lookup := func(id int64) string {
		if len(cache[id]) == 0 {
			user := new(User)
			db.Id(id).Get(user)
			cache[user.Id] = user.Uri
		}
		return cache[id]
	}

	db.Where("name LIKE ?", `%`+query+`%`).Iterate(new(UserName), func(i int, bean interface{}) error {
		elt := bean.(*UserName)
		v := r[lookup(elt.User)]
		v.Name = append(v.Name, elt.Name)
		r[lookup(elt.User)] = v
		return nil
	})

	db.Where("mbox LIKE ?", `%`+query+`%`).Iterate(new(UserMbox), func(i int, bean interface{}) error {
		elt := bean.(*UserMbox)
		v := r[lookup(elt.User)]
		v.Mbox = append(v.Mbox, elt.Mbox)
		r[lookup(elt.User)] = v
		return nil
	})

	for k := range r {
		v := r[k]

		var images []UserImage
		db.Id(k).Find(&images)
		for _, elt := range images {
			v.Image = append(v.Image, elt.Image)
		}
		r[k] = v
	}

	// res1 := make([]User, 0)
	// err := db.Cols("id").Where("uri LIKE ?", `%`+testUser.Uri+`%`).Find(&res1)
	// assert.NoError(t, err)
	// assert.Equal(t, res1[0].Id, testUser.Id)

	// res2 := make([]UserName, 0)
	// err = db.Cols("user").Where("name LIKE ?", `%test%`).Find(&res2)
	// assert.NoError(t, err)
	// assert.Equal(t, res2[0].User, testUser.Id)

	// res3 := make([]UserMbox, 0)
	// err = db.Cols("user").Where("mbox LIKE ?", `%test.com%`).Find(&res3)
	// assert.NoError(t, err)
	// assert.Equal(t, res3[0].User, testUser.Id)

	return
}