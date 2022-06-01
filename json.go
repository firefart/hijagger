package main

type all struct {
	TotalRows int64 `json:"totalrows"`
	Offset    int64 `json:"offset"`
	Rows      []struct {
		ID    string `json:"id"`
		Key   string `json:"key"`
		Value struct {
			Rev string `json:"rev"`
		} `json:"value"`
	} `json:"rows"`
}

type packageJSON struct {
	Name    string `json:"name"`
	NPMUser struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"_npmUser"`
	Maintainers []struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"maintainers"`
}

type downloadJSON struct {
	Downloads int64  `json:"downloads"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Package   string `json:"package"`
}
