package api_handlers

type ApiHandlers struct {
	Board *BoardHandler
	Runs  *RunsHandler
	Repos *ReposHandler
}

func New(boardHandler *BoardHandler, runsHandler *RunsHandler, reposHandler *ReposHandler) *ApiHandlers {
	return &ApiHandlers{
		Board: boardHandler,
		Runs:  runsHandler,
		Repos: reposHandler,
	}
}
