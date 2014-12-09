A probably hazardous shell-based work queue.

(Work in progress.)

# commands

## add

Add a task:

```
goq add cmd arg1 arg2 ...
```

## list

List tasks in a state:

```
goq list <error | waiting | running | stopped>
```

## redo

Redo tasks matching a state:

```
goq redo <error | stopped>
```

Redo a particular task:

```
goq redo <taskid>
```

## Stop

Kill a running task's process.

```
goq kill <signal>
```

Can restart later with `goq redo`

# outputs job id

```
> goq q foobar
12
```

```
> goq wait 12
```

Stream of commands would map to stream of numbers

```
> data | gogo goq q '{{}}'
15
16
17
18
19
20
```

Ask goq to wait on task ids in a stream

```
> taskids | goq wait -
```