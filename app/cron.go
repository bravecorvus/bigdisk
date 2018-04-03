package app

import "time"

func (app *App) RemoveOldRequests() {
	removepasswordresetindexes := make([]int, 0)
	for index, elem := range app.PasswordResets {
		if elem.Expiration.Sub(time.Now()) < 0 {
			removepasswordresetindexes = append(removepasswordresetindexes, index)
		}

	}

	for _, elem := range removepasswordresetindexes {
		app.PasswordResets = append(app.PasswordResets[0:elem], app.PasswordResets[elem+1:]...)
	}

	pendingadminsindexes := make([]int, 0)

	for index, elem := range app.PendingAdmins {
		if elem.Expiration.Sub(time.Now()) < 0 {
			pendingadminsindexes = append(pendingadminsindexes, index)
		}

	}

	for _, elem := range pendingadminsindexes {
		app.PendingAdmins = append(app.PendingAdmins[0:elem], app.PendingAdmins[elem+1:]...)
	}

}
