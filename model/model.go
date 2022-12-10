package model

type PostId string
type UserId string
type ISOTimestamp string
type PageToken string

type Post struct {
	Id        PostId `json:"id,omitempty" pattern:"[A-Za-z0-9_\\-]+"`
	Text      string
	AuthorId  UserId       `json:"authorId,omitempty" pattern:"[0-9a-f]+"`
	CreatedAt ISOTimestamp `json:"createdAt,omitempty" pattern:"\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(\\.\\d{1,3})?Z"`
}

const EmptyPage = PageToken("none")
