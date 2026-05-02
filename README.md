# aimini

Go client for the `ai-images` queue API.

The package is public-safe by design: API URL, token, bot node id, user id, and
input photos are always provided by the application at runtime.

## Install

```bash
go get github.com/elum-utils/aimini
```

## Queue API

```go
client, err := aimini.NewClient(ctx, &aimini.ClientConfig{
	BaseURL: os.Getenv("AIMINI_BASE_URL"),
	Token:   os.Getenv("AIMINI_TOKEN"),
	NodeID:  os.Getenv("AIMINI_NODE_ID"),
})
if err != nil {
	return err
}

item, err := client.Queue.Add(ctx, &aimini.AddQueueItemRequest{
	Image:   aimini.ImageFromFile("/path/to/photo.jpg"),
	UserID:  "123456789",
	NodeID:  "987654321",
	Prompts: []string{"Use the person from Figure 1..."},
})
if err != nil {
	return err
}

_ = item.ID
```

## Model-style API

```go
resp, err := client.Models.GenerateContent(
	ctx,
	"aimini-image",
	[]*aimini.Content{aimini.NewContent(
		aimini.Image(aimini.ImageFromFile("/path/to/photo.jpg")),
		aimini.Text("Use the person from Figure 1..."),
	)},
	&aimini.GenerateContentConfig{
		UserID: "123456789",
	},
)
if err != nil {
	return err
}

fmt.Println(resp.OutputURL)
```

By default, `GenerateContent` waits for the created item to appear in
`queue.list` with `status=processed`. It does not delete the item unless
`DeleteAfter` is set. For Telegram delivery flows, delete only after the image
was successfully sent to the user.

## Local Integration Tests

Keep integration tests outside git. This repository ignores
`integration_local_test.go`, `*_local_integration_test.go`, `.env*`, and local
image files. A local test can read `AIMINI_BASE_URL`, `AIMINI_TOKEN`,
`AIMINI_NODE_ID`, `AIMINI_USER_ID`, and `AIMINI_IMAGE_PATH` from your shell.
