// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package modfetch

import (
	"bytes"
	"strings"
	"testing"

	"cmd/go/internal/modconv"
	"cmd/go/internal/modfile"
	"cmd/go/internal/webtest"
)

func TestConvertLegacyConfig(t *testing.T) {
	webtest.LoadOnce("testdata/webtest.txt")
	webtest.Hook()
	defer webtest.Unhook()

	var tests = []struct {
		path  string
		vers  string
		gomod string
	}{
		{
			// Gopkg.lock parsing.
			"github.com/golang/dep", "v0.4.0",
			`module "github.com/golang/dep"

			require (
				"github.com/Masterminds/semver" v0.0.0-20170726230514-a93e51b5a57e
				"github.com/Masterminds/vcs" v1.11.1
				"github.com/armon/go-radix" v0.0.0-20160115234725-4239b77079c7
				"github.com/boltdb/bolt" v1.3.1
				"github.com/go-yaml/yaml" v0.0.0-20170407172122-cd8b52f8269e
				"github.com/golang/protobuf" v0.0.0-20170901042739-5afd06f9d81a
				"github.com/jmank88/nuts" v0.3.0
				"github.com/nightlyone/lockfile" v0.0.0-20170707060451-e83dc5e7bba0
				"github.com/pelletier/go-toml" v0.0.0-20171218135716-b8b5e7696574
				"github.com/pkg/errors" v0.8.0
				"github.com/sdboyer/constext" v0.0.0-20170321163424-836a14457353
				"golang.org/x/net" v0.0.0-20170828231752-66aacef3dd8a
				"golang.org/x/sync" v0.0.0-20170517211232-f52d1811a629
				"golang.org/x/sys" v0.0.0-20170830134202-bb24a47a89ea
			)`,
		},

		// TODO: https://github.com/docker/distribution uses vendor.conf

		{
			// Godeps.json parsing.
			// TODO: Should v2.0.0 work here too?
			"github.com/docker/distribution", "v0.0.0-20150410205453-85de3967aa93",
			`module "github.com/docker/distribution"
			
			require (
    				"github.com/AdRoll/goamz" v0.0.0-20150130162828-d3664b76d905
				"github.com/bugsnag/bugsnag-go" v0.0.0-20141110184014-b1d153021fcd
				"github.com/bugsnag/osext" v0.0.0-20130617224835-0dd3f918b21b
				"github.com/bugsnag/panicwrap" v0.0.0-20141110184334-e5f9854865b9
				"github.com/docker/libtrust" v0.0.0-20150114040149-fa567046d9b1
				"github.com/garyburd/redigo" v0.0.0-20150301180006-535138d7bcd7
				"github.com/gorilla/context" v0.0.0-20140604161150-14f550f51af5
				"github.com/gorilla/handlers" v0.0.0-20140825150757-0e84b7d810c1
				"github.com/gorilla/mux" v0.0.0-20140926153814-e444e69cbd2e
				"github.com/jlhawn/go-crypto" v0.0.0-20150401213827-cd738dde20f0
				"github.com/yvasiyarov/go-metrics" v0.0.0-20140926110328-57bccd1ccd43
				"github.com/yvasiyarov/gorelic" v0.0.0-20141212073537-a9bba5b9ab50
				"github.com/yvasiyarov/newrelic_platform_go" v0.0.0-20140908184405-b21fdbd4370f
				"golang.org/x/net" v0.0.0-20150202051010-1dfe7915deaf
				"gopkg.in/check.v1" v0.0.0-20141024133853-64131543e789
				"gopkg.in/yaml.v2" v0.0.0-20150116202057-bef53efd0c76
			)`,
		},
	}

	for _, tt := range tests {
		t.Run(strings.Replace(tt.path, "/", "_", -1)+"_"+tt.vers, func(t *testing.T) {
			f, err := modfile.Parse("golden", []byte(tt.gomod), nil)
			if err != nil {
				t.Fatal(err)
			}
			want, err := f.Format()
			if err != nil {
				t.Fatal(err)
			}
			repo, err := Lookup(tt.path)
			if err != nil {
				t.Fatal(err)
			}

			out, err := repo.GoMod(tt.vers)
			if err != nil {
				t.Fatal(err)
			}
			prefix := modconv.Prefix + "\n"
			if !bytes.HasPrefix(out, []byte(prefix)) {
				t.Fatalf("go.mod missing prefix %q:\n%s", prefix, out)
			}
			out = out[len(prefix):]
			if !bytes.Equal(out, want) {
				t.Fatalf("final go.mod:\n%s\n\nwant:\n%s", out, want)
			}
		})
	}
}
