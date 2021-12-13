package clients

// Defines the tweeter type of the tweet.
type TweetType int

// See here for the difference: https://lucid.app/lucidchart/a2666fe4-d9a4-48eb-9f7d-67358427f0d4/edit?invitationId=inv_fad004c2-75b2-4432-85bb-a236c677bcef
const (
	TWEET_TYPE_UNKNOWN = 0
	// When user just created a single tweet.
	TWEET_TYPE_SINGLE = 1
	// When user quote other's tweet (single tweet), and added some comment.
	TWEET_TYPE_QUOTE = 2
	// When user reply to self, creating a chain of twitter telling a long story.
	TWEET_TYPE_THREAD = 3
	// The lightweight way of sharing a tweet to you own homepage.
	TWEET_TYPE_RETWEET = 4
)

const (
	REFERENCE_TYPE_QUOTED = "quoted"

	REFERENCE_TYPE_RETWEETED = "retweeted"

	REFERENCE_TYPE_REPLIED_TO = "replied_to"
)

type Entity struct {
	Hashtags     []interface{} `json:"hashtags"`
	Symbols      []interface{} `json:"symbols"`
	UserMentions []struct {
		ScreenName string `json:"screen_name"`
		Name       string `json:"name"`
		ID         int    `json:"id"`
		IDStr      string `json:"id_str"`
		Indices    []int  `json:"indices"`
	} `json:"user_mentions"`
	Urls  []interface{} `json:"urls"`
	Media []struct {
		ID            int64  `json:"id"`
		IDStr         string `json:"id_str"`
		Indices       []int  `json:"indices"`
		MediaURL      string `json:"media_url"`
		MediaURLHTTPS string `json:"media_url_https"`
		URL           string `json:"url"`
		DisplayURL    string `json:"display_url"`
		ExpandedURL   string `json:"expanded_url"`
		Type          string `json:"type"`
		Sizes         struct {
			Thumb struct {
				W      int    `json:"w"`
				H      int    `json:"h"`
				Resize string `json:"resize"`
			} `json:"thumb"`
			Large struct {
				W      int    `json:"w"`
				H      int    `json:"h"`
				Resize string `json:"resize"`
			} `json:"large"`
			Small struct {
				W      int    `json:"w"`
				H      int    `json:"h"`
				Resize string `json:"resize"`
			} `json:"small"`
			Medium struct {
				W      int    `json:"w"`
				H      int    `json:"h"`
				Resize string `json:"resize"`
			} `json:"medium"`
		} `json:"sizes"`
	} `json:"media"`
}

type User struct {
	ID          int         `json:"id"`
	IDStr       string      `json:"id_str"`
	Name        string      `json:"name"`
	ScreenName  string      `json:"screen_name"`
	Location    string      `json:"location"`
	Description string      `json:"description"`
	URL         interface{} `json:"url"`
	Entities    struct {
		Description struct {
			Urls []interface{} `json:"urls"`
		} `json:"description"`
	} `json:"entities"`
	Protected                      bool   `json:"protected"`
	FollowersCount                 int    `json:"followers_count"`
	FriendsCount                   int    `json:"friends_count"`
	ListedCount                    int    `json:"listed_count"`
	CreatedAt                      string `json:"created_at"`
	FavouritesCount                int    `json:"favourites_count"`
	GeoEnabled                     bool   `json:"geo_enabled"`
	Verified                       bool   `json:"verified"`
	StatusesCount                  int    `json:"statuses_count"`
	ContributorsEnabled            bool   `json:"contributors_enabled"`
	IsTranslator                   bool   `json:"is_translator"`
	IsTranslationEnabled           bool   `json:"is_translation_enabled"`
	ProfileBackgroundColor         string `json:"profile_background_color"`
	ProfileBackgroundImageURL      string `json:"profile_background_image_url"`
	ProfileBackgroundImageURLHTTPS string `json:"profile_background_image_url_https"`
	ProfileBackgroundTile          bool   `json:"profile_background_tile"`
	ProfileImageURL                string `json:"profile_image_url"`
	ProfileImageURLHTTPS           string `json:"profile_image_url_https"`
	ProfileBannerURL               string `json:"profile_banner_url"`
	ProfileLinkColor               string `json:"profile_link_color"`
	ProfileSidebarBorderColor      string `json:"profile_sidebar_border_color"`
	ProfileSidebarFillColor        string `json:"profile_sidebar_fill_color"`
	ProfileTextColor               string `json:"profile_text_color"`
	ProfileUseBackgroundImage      bool   `json:"profile_use_background_image"`
	HasExtendedProfile             bool   `json:"has_extended_profile"`
	DefaultProfile                 bool   `json:"default_profile"`
	DefaultProfileImage            bool   `json:"default_profile_image"`
	TranslatorType                 string `json:"translator_type"`
}

type UserTimelineResponse struct {
	CreatedAt            string `json:"created_at"`
	ID                   int64  `json:"id"`
	IDStr                string `json:"id_str"`
	Text                 string `json:"text"`
	Truncated            bool   `json:"truncated"`
	Entities             Entity `json:"entities,omitempty"`
	Source               string `json:"source"`
	InReplyToStatusID    int64  `json:"in_reply_to_status_id"`
	InReplyToStatusIDStr string `json:"in_reply_to_status_id_str"`
	InReplyToUserID      int    `json:"in_reply_to_user_id"`
	InReplyToUserIDStr   string `json:"in_reply_to_user_id_str"`
	InReplyToScreenName  string `json:"in_reply_to_screen_name"`
	User                 struct {
	} `json:"user"`
	IsQuoteStatus     bool   `json:"is_quote_status"`
	RetweetCount      int    `json:"retweet_count"`
	FavoriteCount     int    `json:"favorite_count"`
	Favorited         bool   `json:"favorited"`
	Retweeted         bool   `json:"retweeted"`
	Lang              string `json:"lang"`
	ExtendedEntities  Entity `json:"extended_entities,omitempty"`
	PossiblySensitive bool   `json:"possibly_sensitive,omitempty"`
}

type UserTimelineResponses []UserTimelineResponse
