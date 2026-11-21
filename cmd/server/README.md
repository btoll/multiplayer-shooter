# Paths

```
main:
    - go handleConnection
        +  game.register
            - g.gameboard.register
            - g.listen

main:
    - game.update
        + g.gameboard.update
            - g.unregister
    - game.render
```

## Lock Boundaries

- Do not have any locks in `gamepiece`.

## Misc

- Unregistering should **only** occur in `gameboard.update`.

