package garminapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGroupsEndpoints(t *testing.T) {
	testEndpoints(t, groupsEndpoints, []endpointCase{
		{"groups.list", map[string]string{"displayName": "tester"}, "GET /group-service/groups/tester?includeMembership=true"},
		{"groups.list", map[string]string{"displayName": "a b/c"}, "GET /group-service/groups/a%20b%2Fc?includeMembership=true"},
		{"groups.search", nil, "POST /group-service/keyword"},
		{"groups.get", map[string]string{"groupID": "1001"}, "GET /group-service/group/1001?includeMembership=true"},
		{"groups.members", map[string]string{"groupID": "1001"}, "GET /group-service/group/1001/members?supportOwner=true&orderByRelationship=true"},
		{"groups.feed", map[string]string{"groupID": "1001"}, "GET /web-gateway/group/v2/1001/groupFeed"},
		{"groups.activities", map[string]string{"groupID": "1001"}, "GET /web-gateway/group/v2/1001/activityFeed"},
		{"groups.challenges", map[string]string{"groupID": "1001"}, "GET /groupchallenge-service/group/1001/groupchallenge/nonCompleted"},
		{"groups.leaderboard", map[string]string{"userType": "activity", "groupID": "1001", "start": "1", "limit": "20", "todayDate": "2026-10-21", "startDate": "2026-10-19", "endDate": "2026-10-25", "metricID": "17", "actTypeID": "1"}, "GET /groupchallenge-service/leaderboard/activity/group/1001?start=1&limit=20&todayDate=2026-10-21&startDate=2026-10-19&endDate=2026-10-25&metricId=17&actTypeId=1"},
		{"groups.leaderboard", map[string]string{"userType": "wellness", "groupID": "1001", "start": "3", "limit": "2", "todayDate": "2026-10-21", "startDate": "2026-10-19", "endDate": "2026-10-25", "metricID": "29"}, "GET /groupchallenge-service/leaderboard/wellness/group/1001?start=3&limit=2&todayDate=2026-10-21&startDate=2026-10-19&endDate=2026-10-25&metricId=29"},
		{"groups.requests", map[string]string{"displayName": "tester"}, "GET /group-service/private/group/request/user/tester"},
		{"groups.join", map[string]string{"groupID": "1001", "profileID": "3141592"}, "POST /group-service/group/1001/member/3141592"},
		{"groups.leave", map[string]string{"groupID": "1001", "profileID": "3141592"}, "DELETE /group-service/group/1001/member/3141592"},
		{"groups.create", nil, "POST /group-service/group"},
		{"groups.update", map[string]string{"groupID": "1001"}, "PUT /group-service/group/1001"},
		{"groups.delete", map[string]string{"groupID": "1001"}, "DELETE /group-service/group/1001"},
	})
}

const savedGroup = `{"id":1001,"groupName":"Old","groupDescription":"Old text","ownerId":2718281,"defaultImageName":"running_01","profileImageUrlLarge":null,"groupVisibility":"VISIBLE","groupPrivacy":"PUBLIC","location":null,"primaryActivities":["RUNNING"],"leaderboardTypes":[],"membershipStatus":"OWNER"}`

func str(s string) *string { return &s }

func groupServer(t *testing.T, saved string) (*Client, *map[string]any) {
	t.Helper()
	sent := map[string]any{}
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, saved)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Errorf("%s %s: body is not a JSON object: %v", r.Method, r.URL.Path, err)
		}
		_, _ = io.WriteString(w, `{"id":1001}`)
	}))
	return client, &sent
}

func TestUpdateGroup_SendsTheFullObjectWithTheChangedFields(t *testing.T) {
	client, sent := groupServer(t, savedGroup)
	_, err := client.UpdateGroup(context.Background(), "1001", GroupEdit{Description: str("New text"), Location: str("Berlin")})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"id": float64(1001), "groupName": "Old", "groupDescription": "New text", "ownerId": float64(2718281),
		"defaultImageName": "running_01", "profileImageUrlLarge": nil, "groupVisibility": "VISIBLE",
		"groupPrivacy": "PUBLIC", "location": "Berlin", "primaryActivities": []any{"RUNNING"},
		"leaderboardTypes": []any{}, "membershipStatus": "OWNER",
	}
	gotJSON, _ := json.Marshal(*sent)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("PUT body:\n got %s\nwant %s", gotJSON, wantJSON)
	}
}

func TestUpdateGroup_EmptyTextClearsTheField(t *testing.T) {
	client, sent := groupServer(t, savedGroup)
	if _, err := client.UpdateGroup(context.Background(), "1001", GroupEdit{Description: str("")}); err != nil {
		t.Fatal(err)
	}
	if v, ok := (*sent)["groupDescription"]; !ok || v != "" {
		t.Errorf("groupDescription = %#v, want an empty string", v)
	}
}

func TestUpdateGroup_HiddenGroupIsInviteOnly(t *testing.T) {
	tests := []struct {
		name      string
		saved     string
		edit      GroupEdit
		wantErr   string
		wantVis   string
		wantPriv  string
		wantWrite bool
	}{
		{"hiding sets invite", savedGroup, GroupEdit{Visibility: str("HIDDEN")}, "", "HIDDEN", "INVITE", true},
		{"hiding with invite", savedGroup, GroupEdit{Visibility: str("HIDDEN"), Privacy: str("INVITE")}, "", "HIDDEN", "INVITE", true},
		{"hiding with public is refused", savedGroup, GroupEdit{Visibility: str("HIDDEN"), Privacy: str("PUBLIC")}, "invite-only", "", "", false},
		{"public on a hidden group is refused", strings.Replace(strings.Replace(savedGroup, `"VISIBLE"`, `"HIDDEN"`, 1), `"PUBLIC"`, `"INVITE"`, 1), GroupEdit{Privacy: str("PUBLIC")}, "invite-only", "", "", false},
		{"showing a hidden group keeps its privacy", strings.Replace(strings.Replace(savedGroup, `"VISIBLE"`, `"HIDDEN"`, 1), `"PUBLIC"`, `"INVITE"`, 1), GroupEdit{Visibility: str("VISIBLE")}, "", "VISIBLE", "INVITE", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, sent := groupServer(t, tc.saved)
			_, err := client.UpdateGroup(context.Background(), "1001", tc.edit)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want one containing %q", err, tc.wantErr)
				}
				if len(*sent) != 0 {
					t.Errorf("sent a body after the refusal: %v", *sent)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if (*sent)["groupVisibility"] != tc.wantVis || (*sent)["groupPrivacy"] != tc.wantPriv {
				t.Errorf("visibility/privacy = %v/%v, want %s/%s", (*sent)["groupVisibility"], (*sent)["groupPrivacy"], tc.wantVis, tc.wantPriv)
			}
		})
	}
}

func TestUpdateGroup_NotAnObject(t *testing.T) {
	client, _ := groupServer(t, `[]`)
	if _, err := client.UpdateGroup(context.Background(), "1001", GroupEdit{Name: str("New")}); err == nil {
		t.Fatal("want an error for a body that is not an object")
	}
}

func TestCreateGroup_SendsTheFormFields(t *testing.T) {
	client, sent := groupServer(t, "")
	_, err := client.CreateGroup(context.Background(), NewGroup{Name: "Club", Visibility: "HIDDEN", Privacy: "INVITE", Image: "running_01"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"activityFeedTypes":[],"defaultImageName":"running_01","facebookUrl":"","groupDescription":"","groupName":"Club","groupPrivacy":"INVITE","groupVisibility":"HIDDEN","leaderboardTypes":[],"location":"","otherPrimaryActivity":"","featureTypes":[],"primaryActivities":[],"profileImageUrlLarge":null,"profileImageUrlMedium":null,"profileImageUrlSmall":null,"twitterUrl":"","websiteUrl":""}`
	var wantMap map[string]any
	_ = json.Unmarshal([]byte(want), &wantMap)
	gotJSON, _ := json.Marshal(*sent)
	wantJSON, _ := json.Marshal(wantMap)
	if string(gotJSON) != string(wantJSON) {
		t.Errorf("POST body:\n got %s\nwant %s", gotJSON, wantJSON)
	}
}

func TestGroupsSearchIsAForm(t *testing.T) {
	var contentType, body string
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		_, _ = w.Write([]byte(`{}`))
	}))
	if _, err := client.Call(context.Background(), "groups.search", nil, []byte("keyword=running&limit=20&start=1")); err != nil {
		t.Fatal(err)
	}
	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", contentType)
	}
	if body != "keyword=running&limit=20&start=1" {
		t.Errorf("body = %q, want the form as given", body)
	}
}
