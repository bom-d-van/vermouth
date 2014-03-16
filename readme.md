## Vermouth

Vermouth is a small tool designed for summarizing the changes in a go package in different versions.

Vermouth is very simple to use. Supposed you have package which named `github.com/bom-d-van/pkg`, what you need to do is:

```
cd ~GOPATH/src/github.com/bom-d-van
git clone pkg oldpkg
cd oldpkg
git reset $checksum-pointing-to-a-old-version
vermouth -prev=github.com/bom-d-van/oldpkg -new=github.com/bom-d-van/pkg
```

Then you could see some changes print out like this:

```
The New API is NOT backward compatible.
Structs:
Modified:
User:
	Type Changes:
		ProfileURL: string -> template.HTMLAttr
	New Fields:
		HasMobileDevices bool
TaskLog:
	New Fields:
		IsDueChanged bool
		LocalOldDue string
		LocalNewDue string
Task:
	New Fields:
		LocalDueWithYear string
		CanEditDueDate bool
		FarAwayCorner bool
TaskOutline:
	Deprecated Fields:
		EntryId bson.ObjectId
	Type Changes:
		RootId: bson.ObjectId -> string
	New Fields:
		CommentId string
EntryInput:
	New Fields:
		UploadGroupId string
		BasedPostLangCode string
TaskInput:
	New Fields:
		TaskDue string
		EntryId string
========
Interfaces:
Modified:
AuthUserService:
	Signature Changes:
		UpdateTask(input *TaskInput) (task *Task, err error)
		-> UpdateTask(taskInput *TaskInput) (task *Task, err error)
	New Methods:
		UpdateSimpleTask(input *TaskInput) (task *Task, err error)
		RegisterAppleDeviceForUser(userId string, token string) (err error)
		UnregisterAppleDeviceForUser(userId string, token string) (err error)
```

Vermouth is not designed to replace human-edited change logs completely. On the contrary, by the using of this small tool, we could save some time to add more useful stuff in our package change logs.

## Notes

So far, vermouth only compare structs and interfaces, in which only fields in structs will be took into account. More changes could be expected to be supported if this tool is proved to be a good and useful one.
