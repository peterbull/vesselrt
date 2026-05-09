package main

import (
	_ "context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

func testFunc() int {
	return 123
}
func testFunc2() {
	fmt.Println("called testfunc2")
}
func DoWork(i int) int {
	fmt.Printf("%v", i)
	return i
}
func oldWg() {
	dataChan := make(chan int)
	testFunc()
	testFunc2()
	go func() {
		wg := sync.WaitGroup{}
		for i := range 1000 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dataChan <- DoWork(i)
			}()
		}
		wg.Wait()
		close(dataChan)
	}()
	for i := range dataChan {
		fmt.Printf("datachan i: %v\n", i)
	}
}
func newWg() {
	dataChan := make(chan int)
	var (
		wg sync.WaitGroup
	)

	go func() {
		for i := range 1000 {
			wg.Go(func() {
				dataChan <- DoWork(i)
			})
		}
		wg.Wait()
		close(dataChan)
	}()

	for i := range dataChan {
		fmt.Printf("datachan i: %v\n", i)
	}
}

type AuthResponse struct {
	Token string `json:"token"`
}

type ManifestList struct {
	Manifests []ManifestEntry `json:"manifests"`
}
type ManifestEntry struct {
	Digest    string   `json:"digest"`
	MediaType string   `json:"mediaType"`
	Platform  Platform `json:"platform"`
}

type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

func fetchDockerToken() (AuthResponse, error) {
	res, err := http.Get("https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/alpine:pull")
	if err != nil {
		return AuthResponse{}, fmt.Errorf("fetching docker token: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return AuthResponse{}, fmt.Errorf("unexpected status: %s", res.Status)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(res.Body).Decode(&authResp); err != nil {
		return AuthResponse{}, fmt.Errorf("decoding auth response: %w", err)
	}

	return authResp, nil
}
func fetchArmImage(token string, arch string) (ManifestEntry, error) {
	req, _ := http.NewRequest("GET", "https://registry-1.docker.io/v2/library/alpine/manifests/latest", nil)

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return ManifestEntry{}, fmt.Errorf("error getting manifest: %v", err)
	}

	defer res.Body.Close()
	final, err := io.ReadAll(res.Body)
	if err != nil {
		return ManifestEntry{}, fmt.Errorf("error reading manifest: %v", err)
	}
	// var result map[string]any
	var result ManifestList
	if err := json.Unmarshal(final, &result); err != nil {
		return ManifestEntry{}, fmt.Errorf("error parsing manifest: %v", err)
	}

	fmt.Printf("result: %v", result)
	fmt.Println("break")
	for _, manifest := range result.Manifests {
		if manifest.Platform.Architecture == arch {
			return manifest, nil
		}
	}
	return ManifestEntry{}, fmt.Errorf("no matching arm images found")

}

func main() {
	fmt.Println("scratch")
	// // oldWg()
	//
	// newWg()
	// var (
	// 	reader io.Reader
	// )
	// req, err := http.NewRequestWithContext(context.Background(), "GET", "https://www.google.com", reader)
	// if err != nil {
	// 	fmt.Printf("error: %v", err)
	// }
	// res, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	fmt.Printf("err: %v", err)
	// }
	// fmt.Printf("res: %v", res)

	authResp, err := fetchDockerToken()
	if err != nil {
		log.Fatalf("somthing happened and you aint get no token: %v", err)
	}
	manifest, err := fetchArmImage(authResp.Token, "arm64")
	if err != nil {
		log.Fatalf("something happened and no manifest lists: %v", err)
	}
	fmt.Printf("manifest list: %v", manifest)

	fmt.Println("break")
}
