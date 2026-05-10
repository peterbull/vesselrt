package hub

import (
	"archive/tar"
	"compress/gzip"
	_ "context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
)

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

type ImageManifest struct {
	SchemaVersion int        `json:"schemaVersion"`
	MediaType     string     `json:"mediaType"`
	Config        LayerRef   `json:"config"`
	Layers        []LayerRef `json:"layers"`
}

type LayerRef struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
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

func fetchImageManifest(token string, arch string) (ImageManifest, error) {
	req, _ := http.NewRequest("GET", "https://registry-1.docker.io/v2/library/alpine/manifests/latest", nil)

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	client := &http.Client{}
	res, err := client.Do(req)

	if err != nil {
		return ImageManifest{}, fmt.Errorf("error getting manifest: %v", err)
	}

	defer res.Body.Close()
	final, err := io.ReadAll(res.Body)
	if err != nil {
		return ImageManifest{}, fmt.Errorf("error reading manifest: %v", err)
	}
	var result ManifestList
	if err := json.Unmarshal(final, &result); err != nil {
		return ImageManifest{}, fmt.Errorf("error parsing manifest: %v", err)
	}

	idx := slices.IndexFunc(result.Manifests, func(m ManifestEntry) bool {
		return m.Platform.Architecture == arch
	})

	var targetManifest *ManifestEntry

	if idx != -1 {
		targetManifest = &result.Manifests[idx]
	}

	if targetManifest == nil {
		return ImageManifest{}, fmt.Errorf("no matching arm images found")
	}

	req, _ = http.NewRequest("GET", fmt.Sprintf("https://registry-1.docker.io/v2/library/alpine/manifests/%s", targetManifest.Digest), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	res, err = client.Do(req)
	if err != nil {
		return ImageManifest{}, fmt.Errorf("error getting manifest: %v", err)
	}
	final, err = io.ReadAll(res.Body)
	if err != nil {
		return ImageManifest{}, fmt.Errorf("error reading manifest: %v", err)
	}

	defer res.Body.Close()
	var imageResult ImageManifest
	if err := json.Unmarshal(final, &imageResult); err != nil {
		return ImageManifest{}, fmt.Errorf("error parsing manifest: %v", err)
	}

	return imageResult, nil
}

func fetchLayer(token string, imageName string, digest string) error {
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://registry-1.docker.io/v2/library/%s/blobs/%s", imageName, digest), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.oci.image.layer.v1.tar+gzip")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error getting layer: %v", err)
	}
	defer res.Body.Close()

	destDir := fmt.Sprintf("images/%s/rootfs", imageName)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %v", err)
	}

	return unpackLayer(res.Body, destDir)
}

func unpackLayer(r io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %v", err)
		}

		target := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, os.FileMode(header.Mode))
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			f, _ := os.Create(target)
			io.Copy(f, tr)
			f.Close()
		case tar.TypeSymlink:
			os.Symlink(header.Linkname, target)
		}
	}
	return nil
}

func PullImage(imageName string) {
	authResp, err := fetchDockerToken()
	if err != nil {
		log.Fatalf("somthing happened and you aint get no token: %v", err)
	}
	manifest, err := fetchImageManifest(authResp.Token, "arm64")
	if err != nil {
		log.Fatalf("something happened and no manifest lists: %v", err)
	}
	for _, layer := range manifest.Layers {
		if err := fetchLayer(authResp.Token, imageName, layer.Digest); err != nil {
			log.Fatalf("something happened and no layers all messed up and stuff: %v", err)
		}
	}
}
