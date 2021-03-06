/*
 * Copyright 2016 Fabrício Godoy
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mongostore

import (
	"fmt"
	"testing"
	"time"

	"github.com/raiqub/data/testdata"
	"github.com/skarllot/raiqub/test"
	"gopkg.in/mgo.v2"
	"gopkg.in/raiqub/dot.v1"
)

const (
	colName     = "expire"
	mongoURLTpl = "mongodb://%s:%d/raiqub"
)

type TestLogger struct {
	t *testing.T
}

func (l TestLogger) Output(calldepth int, s string) error {
	l.t.Log(s)
	return nil
}

func TestMongoStore(t *testing.T) {
	session, env := prepareMongoEnvironment(t)
	defer env.Dispose()
	mgo.SetLogger(TestLogger{t})

	//	session, err := openSession("mongodb://localhost/raiqub")
	//	if err != nil {
	//		t.Fatalf("Error opening a MongoDB session: %s\n", err)
	//	}
	//	defer session.Close()

	store := New(session.DB(""), colName, time.Millisecond)
	store.EnsureAccuracy(true)

	testdata.TestAtomic(store, t)

	store.Flush()
	testdata.TestExpiration(store, t)

	store.Flush()
	testdata.TestValueHandling(store, t)

	store.Flush()
	testdata.TestKeyCollision(store, t)

	//store.Flush()
	//testdata.TestSetExpiration(store, t)

	store.Flush()
	testdata.TestPostpone(store, t)

	store.Flush()
	testdata.TestTransient(store, t)

	store.Flush()
	testdata.TestTypeError(store, t)
}

func BenchmarkMongoStoreAddGet(b *testing.B) {
	session, env := prepareMongoEnvironment(b)
	defer env.Dispose()

	store := New(session.DB(""), colName, time.Second)
	testdata.BenchmarkAddGet(store, b)
}

func BenchmarkMongoStoreAddGetTransient(b *testing.B) {
	session, env := prepareMongoEnvironment(b)
	defer env.Dispose()

	store := New(session.DB(""), colName, time.Second)
	store.SetTransient(true)
	testdata.BenchmarkAddGet(store, b)
}

func BenchmarkMongoAtomicIncrement(b *testing.B) {
	session, env := prepareMongoEnvironment(b)
	defer env.Dispose()

	store := New(session.DB(""), colName, time.Second)
	store.SetTransient(true)
	testdata.BenchmarkAtomicIncrement(store, b)
}

func openSession(url string) (*mgo.Session, error) {
	session, err := mgo.Dial(url)
	if err != nil {
		return nil, err
	}

	session.SetMode(mgo.Monotonic, true)

	_, err = session.DB("").CollectionNames()
	if err != nil {
		return nil, err
	}

	return session, nil
}

func prepareMongoEnvironment(tb testing.TB) (*mgo.Session, dot.Disposable) {
	env := dot.NewMulticastDispose()
	mongo := test.NewMongoDBEnvironment(tb)
	if !mongo.Applicability() {
		tb.Skip("This test cannot be run because Docker is not acessible")
	}

	if !mongo.Run() {
		tb.Fatal("Could not start MongoDB server")
	}
	env.Add(func() {
		mongo.Stop()
	})

	net, err := mongo.Network()
	if err != nil {
		env.Dispose()
		tb.Fatalf("Error getting MongoDB IP address: %s\n", err)
	}

	mgourl := fmt.Sprintf(mongoURLTpl, net[0].IpAddress, net[0].Port)

	session, err := openSession(mgourl)
	if err != nil {
		env.Dispose()
		tb.Fatalf("Error opening a MongoDB session: %s\n", err)
	}
	env.Add(func() {
		session.Close()
	})

	return session, env
}
