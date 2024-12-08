### This is a very basic and limited thingy to do something with a png file 
 it can take a png file as input and output it as a ppm file 
 it was made as and effort to learn golang and handling media files and png images

## current status

- 8 bit RGBA png only
- no interlacing methods supported
- no trasnparency
- only outputs to ppm 
- very very slow


### how to try it out
- clone the repo
- run `go run main.go <your input png>`
- inspect the `output.ppm` file with your desired image viewer


### resources
- [stb_image header](https://github.com/nothings/stb/blob/master/stb_image.h)
- [PNG wiki](https://en.wikipedia.org/wiki/PNG)
- [PNG spec](http://www.libpng.org/pub/png/spec/1.2/PNG-Contents.html)
- [Sphareophoria youtube playlist](https://www.youtube.com/playlist?list=PL980gcR1LE3Kcujd_vAtZSvO92a44xBcW)

