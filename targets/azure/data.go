package azure_target

import "time"

type pushRef struct {
	Name        string `json:"name"`
	OldObjectID string `json:"oldObjectId"`
	NewObjectID string `json:"newObjectId,omitempty"`
}

type pushItem struct {
	Path string `json:"path"`
}

type pushContent struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
}

type commitChange struct {
	ChangeType string      `json:"changeType"`
	Item       pushItem    `json:"item"`
	NewContent pushContent `json:"newContent"`
}

type commitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

type commit struct {
	Comment   string         `json:"comment"`
	Changes   []commitChange `json:"changes"`
	CommitID  string         `json:"commitId,omitempty"`
	Author    commitAuthor   `json:"author,omitempty"`
	Committer commitAuthor   `json:"committer,omitempty"`
	URL       string         `json:"url,omitempty"`
}

type pushData struct {
	RefUpdates []pushRef `json:"refUpdates"`
	Commits    []commit  `json:"commits"`
}

type azureRef struct {
	Name     string `json:"name"`
	ObjectID string `json:"objectId"`
}

type refList struct {
	Value []azureRef `json:"value"`
	Count int        `json:"count"`
}

type repository struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	URL     string `json:"url"`
	Project struct {
		ID             string    `json:"id"`
		Name           string    `json:"name"`
		Description    string    `json:"description"`
		URL            string    `json:"url"`
		State          string    `json:"state"`
		Revision       int       `json:"revision"`
		Visibility     string    `json:"visibility"`
		LastUpdateTime time.Time `json:"lastUpdateTime"`
	} `json:"project"`
	Size       int    `json:"size"`
	RemoteURL  string `json:"remoteUrl"`
	WebURL     string `json:"webUrl"`
	IsDisabled bool   `json:"isDisabled"`
}

type pushResponse struct {
	Commits    []commit   `json:"commits"`
	RefUpdates []pushRef  `json:"refUpdates"`
	Repository repository `json:"repository"`
	PushID     int        `json:"pushId"`
	Date       time.Time  `json:"date"`
	URL        string     `json:"url"`
}
