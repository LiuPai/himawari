# himawari
GO version of himawaripy https://github.com/boramalper/himawaripy

himawari is a golang project that fetches near-realtime (10 minutes delayed)
picture of Earth as its taken by
[Himawari 8 (ひまわり8号)](https://en.wikipedia.org/wiki/Himawari_8) and sets it
as your desktop background.

Set a cronjob that runs in every 10 minutes to automatically get the
near-realtime picture of Earth.

## install
```
go get github.com/LiuPai/himawari/...
```

## Usage
You can configure the level of detail, by modifying the parameter. You can set the
parameter `level` to `4`, `8`, `16`, or `20` to increase the quality (and
thus the file size as well). Please keep in mind that it will also take more
time to download the tiles.

You can also change the path of the latest picture, which is by default
`/tmp/himawari.png`, by changing the `output` parameter.
```
	himawari
		-cache string
			Path to the cache file directory (default "/tmp")
		-daemon
			Run himawari as daemon
		-level int
			Image quality and size choose one of [4, 8, 16, 20] (default 4)
		-output string
			The link of current himawari image (default "/tmp/himawari.png")
		-pid string
			Himawari unix like system pid file
		-tick uint
			Duration to check himawari latest timestamp in seconds (default 300)
```

## Configuration
Change your desktop background point to himawari parameter -output(defalut: "/tmp/himawari.png").
Most desktop environment will automatically update when file changed.

If you would like to share why, you can contact me on github.

## Example
![Earth, as 2016/02/04/13:30:00 GMT](http://i.imgur.com/4XA6WaM.jpg)

## TODO
* border line support.
* area select.

## Attributions
Thanks to *[MichaelPote](https://github.com/MichaelPote)* for the [initial
implementation](https://gist.github.com/MichaelPote/92fa6e65eacf26219022) using
Powershell Script.

Thanks to *[Charlie Loyd](https://github.com/celoyd)* for image processing logic
([hi8-fetch.py](https://gist.github.com/celoyd/39c53f824daef7d363db)).

Thanks to *[Bora M. Alper](https://github.com/boramalper)* for the [python version
implementation](https://github.com/boramalper/himawaripy)

Obviously, thanks to the Japan Meteorological Agency for opening these pictures
to public.
