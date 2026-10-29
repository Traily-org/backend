package postgres

import _ "embed"

//go:embed queries/get_user.sql
var getUserQuery string

//go:embed queries/create_user.sql
var createUserQuery string

//go:embed queries/update_user.sql
var updateUserQuery string

//go:embed queries/delete_user.sql
var deleteUserQuery string

//go:embed queries/get_track.sql
var getTrackQuery string

//go:embed queries/list_tracks_by_user.sql
var listTracksByUserQuery string

//go:embed queries/create_track.sql
var createTrackQuery string

//go:embed queries/update_track.sql
var updateTrackQuery string

//go:embed queries/delete_track.sql
var deleteTrackQuery string

// Other repositories keep their SQL inline.
