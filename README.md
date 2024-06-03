# Orchestra

Orchestra is a library to manage long running go processes.

At the heart of the library is an interface called Player

```go
// Player is a long running background worker
type Player interface {
    Play(context.Context) error
}
```

All a type needs to do to satisfy the interface is to have a `Play` method that will gracefully shutdown when the context is done.

It can also return an error if it encounters a problem when playing.

Next, there's the Conductor type (which itself is a Player)

```go
// Conductor is a group of workers. It is also a Player itself **evil laugh**
type Conductor struct {
    Timeout time.Duration
    Players map[string]Player
}
```

With the conductor, you add `Players` to it, and when you call the `Play` method on the conductor, it will start the `Players` under it and gracefully shut them all down when the main context is done.

The timeout is there incase there is a `Player` that refused to stop

## Helper functions

### `PlayUntilSignal(p Player, sig ...os.Signal)`

This will start a player with a context, and close the context once it receives any of the signals provided.

Example:

```go
package main

import (
    "os"
    "syscall"

    "github.com/stephenafamo/orchestra"
)

func main() {
    player := ... // something that satisfies the player interface
    err := orchestra.PlayUntilSignal(player, os.Interrupt, syscall.SIGTERM)
    if err != nil {
        panic(err)
    }
}
```

### `PlayerFunc(func(context.Context) error)`

`PlayerFunc` is a quick way to convert a standalone function into a type that satisfies the `Player` interface.

```go
package main

import (
    "context"
    "os"
    "time"
    "syscall"

    "github.com/stephenafamo/orchestra"
)

func main() {
    player := orchestra.PlayerFunc(myFunction)
    err := orchestra.PlayUntilSignal(player, os.Interrupt, syscall.SIGTERM)
    if err != nil {
        panic(err)
    }
}

func myFunction(ctx context.Context) error {
    // A continuously running process
    // Exits when ctx is done
    <-ctx.Done()
    return nil
}
```

### `NewServerPlayer(*http.Server)`

`ServerPlayer` is a type that embeds the `*http.Server` and extends it to satisfy the `Player` interface.

Since a very common long running process is the `*http.Server`, this makes it easy to create a player from one without having to re-write the boilerplate each time.

With the help of multiple helper functions, we can create a gracefully shutting down server that closes on `SIGINT` and `SIGTERM` by:

```go
package main

import (
    "net/http"
    "os"
    "syscall"

	"github.com/cenkalti/backoff/v4"
    "github.com/stephenafamo/orchestra"
)

func main() {
    s := orchestra.NewServerPlayer(
        // Setting the *http.Server
        &http.Server{Addr: ":8080"},
        // Sets the timeout waiting for the server to stop.
        orchestra.WithShutdownTimeout(time.Second * 5),
        // With TLS makes the server use ListenAndServeTLS
        orchestra.WithTLS(),
        // With Backoff adds a backoff strategy to the server
        orchestra.WithBackoff(backoff.NewExponentialBackOff()),
    )
    err := orchestra.PlayUntilSignal(s, os.Interrupt, syscall.SIGTERM)
    if err != nil {
        panic(err)
    }
}
```

## Using the `Conductor`

The `Conductor` type makes it easy to coordinate multiple long running processes. Because each one is blocking, it is often clumsy to start and stop all of them nicely.

Well, the `Conductor` is here to make the pain go away.

```go
package main

import (
    "context"
    "net/http"
    "os"
    "syscall"
    "time"

    "github.com/stephenafamo/orchestra"
)

func main() {
    // A player from a function
    a := orchestra.PlayerFunc(myFunction)
    // A player from a server
    b := orchestra.NewServerPlayer(myServer)

    // A conductor to control them all
    conductor := &orchestra.Conductor{
        Timeout: 5 * time.Second,
        Players: map[string]orchestra.Player{
            // the names are used to identify the players
            // both in logs and the returned errors
            "function": a,
            "server":   b,
        },
    }

    // Use the conductor as a Player
    err := orchestra.PlayUntilSignal(conductor, os.Interrupt, syscall.SIGTERM)
    if err != nil {
        panic(err)
    }
}

func myFunction(ctx context.Context) error {
    // A continuously running process
    // Exits when ctx is done
    <-ctx.Done()
    return nil
}
```

Note: The Conductor makes sure that if by some mistake you add the conductor as a player to itself (or another conductor under it), it will not start the players multiple times.

If the conductor has to exit because of the timeout and not because all the `Players` exited successfully, it will return an error of type `TimeoutErr`.

You can ignore this type of error by checking for it like this:

```go
// Use the conductor as a Player
err := orchestra.PlayUntilSignal(conductor, os.Interrupt, syscall.SIGTERM)
if err != nil && !errors.As(err, &orchestra.TimeoutErr{}) {
    panic(err)
}
```

Or you can specially handle it like this:

```go
// Use the conductor as a Player
err := orchestra.PlayUntilSignal(conductor, os.Interrupt, syscall.SIGTERM)
if err != nil {
    timeoutErr := orchestra.TimeoutErr{}
    if errors.As(err, &timeoutErr) {
        fmt.Println(timeoutErr) // Handle the timeout error
    } else {
        panic(err) // handle other errors
    }
}
```

### Restarting Players

A player can be configured to restart, possibly with exponential backoff by implementing the `PlayerWithBackoff` interface.

For example:

```go
package main

import "github.com/cenkalti/backoff/v4"

type playerThatRestartsImmediately struct{}

func (playerThatRestartsImmediately) Backoff() backoff.BackOff {
	return &backoff.ZeroBackOff{}
}


type playerWithExponentialBakoff struct{}

func (playerWithExponentialBakoff) Backoff() backoff.BackOff {
	return backoff.NewExponentialBackOff()
}
```

## Customization

The logger can be modified by assigning a logger to `orchestra.Logger`

```go
type Logger interface {
	Info(msg string, attrs ...slog.Attr)
	Error(msg string, attrs ...slog.Attr)
	WithGroup(name string) Logger
}
```

If you have an existing `*slog.Logger`, you can create an `orchestra.Logger` by using the `orchestra.LoggerFromSlog` function.

```go
orchestraLogger := orchestra.LoggerFromSlog(slog.LevelInfo, slog.LevelError, slog.Default())
```

## Contributing

Looking forward to pull requests.
