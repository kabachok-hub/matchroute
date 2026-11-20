package main

import (
	"encoding/json"
	"testing"
)

func TestNewestInstaller(t *testing.T) {
	const fixture = `[
		{"tag_name":"matchroute-v9.0.0","draft":true,"assets":[
			{"name":"MatchRoute-v9.0.0-windows-x64-setup.exe","state":"uploaded","browser_download_url":"https://github.com/kabachok-hub/matchroute/releases/download/matchroute-v9.0.0/MatchRoute-v9.0.0-windows-x64-setup.exe"}]},
		{"tag_name":"matchroute-v0.1.8","assets":[
			{"name":"MatchRoute-v0.1.8-windows-x64-setup.exe","state":"uploaded","browser_download_url":"https://example.org/not-a-github-asset.exe"}]},
		{"tag_name":"matchroute-v0.1.4","assets":[
			{"name":"MatchRoute-v0.1.4-windows-x64-setup.exe","state":"uploaded","browser_download_url":"https://github.com/kabachok-hub/matchroute/releases/download/matchroute-v0.1.4/MatchRoute-v0.1.4-windows-x64-setup.exe"}]},
		{"tag_name":"matchroute-v0.1.5","prerelease":true,"assets":[
			{"name":"MatchRoute-v0.1.5-windows-x64-setup.exe","state":"uploaded","browser_download_url":"https://github.com/kabachok-hub/matchroute/releases/download/matchroute-v0.1.5/MatchRoute-v0.1.5-windows-x64-setup.exe"}]},
		{"tag_name":"matchroute-v0.1.6","assets":[
			{"name":"SHA256SUMS.txt","state":"uploaded","browser_download_url":"https://github.com/kabachok-hub/matchroute/releases/download/matchroute-v0.1.6/SHA256SUMS.txt"}]}
	]`
	var releases []githubRelease
	if err := json.Unmarshal([]byte(fixture), &releases); err != nil {
		t.Fatal(err)
	}
	got := newestInstaller(releases, "0.1.3")
	want := releaseUpdate{
		version: "0.1.5",
		setup:   releaseDownloads + "matchroute-v0.1.5/MatchRoute-v0.1.5-windows-x64-setup.exe",
	}
	if got == nil || *got != want {
		t.Fatalf("update = %+v, want %+v", got, want)
	}
	if got := newestInstaller(releases, "0.1.5"); got != nil {
		t.Fatalf("same version, drafts and incomplete releases must be ignored: %+v", got)
	}
}
