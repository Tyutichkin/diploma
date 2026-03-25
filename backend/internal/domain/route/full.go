package route

type Full struct {
	Route    Route
	Stops    []Stop
	Stats    *Stats    `json:",omitempty"`
	Geometry *Geometry `json:",omitempty"`
}
