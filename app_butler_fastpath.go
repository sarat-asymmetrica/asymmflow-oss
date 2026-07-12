package main

import butlerfastpath "ph_holdings_app/pkg/butler/fastpath"

func (a *App) butlerFastpathService() butlerfastpath.Service {
	return butlerfastpath.Service{
		DB:       a.butlerDatabasePort(),
		Workflow: a.butlerWorkflowPort(),
		AppCtx:   a.butlerAppContext(),
	}
}
