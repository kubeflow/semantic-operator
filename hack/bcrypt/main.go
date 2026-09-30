/*
Copyright The Kubeflow Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Command bcrypt emits a Trino password file from user=password pairs.
//
//	go run ./hack/bcrypt "semantic-manager=manager" "semantic-server=server" > password.db
//
// Trino's file password authenticator expects the bcrypt $2y variant. Go's
// bcrypt emits the equivalent $2a hash; the digest after the version tag is
// identical, so rewriting the prefix to $2y is safe. This keeps engine test
// setup dependency-free (no python, no htpasswd) since the Go toolchain is
// already required to build the operator.
package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// cost matches the rounds used for the engine test users. 10 is Trino's
// documented minimum and is plenty for throwaway test credentials.
const cost = 10

func main() {
	pairs := os.Args[1:]
	if len(pairs) == 0 {
		fmt.Fprintln(os.Stderr, `usage: bcrypt "user=password" ["user=password" ...]`)
		os.Exit(2)
	}
	var b strings.Builder
	for _, p := range pairs {
		user, pw, ok := strings.Cut(p, "=")
		if !ok || user == "" {
			fmt.Fprintf(os.Stderr, "invalid pair %q: want user=password\n", p)
			os.Exit(2)
		}
		h, err := bcrypt.GenerateFromPassword([]byte(pw), cost)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hashing %q: %v\n", user, err)
			os.Exit(1)
		}
		// Rewrite the $2a version tag to the $2y tag Trino requires. The rest
		// of the hash (cost, salt, digest) is byte-identical.
		fmt.Fprintf(&b, "%s:$2y$%s\n", user, string(h)[4:])
	}
	fmt.Print(b.String())
}
