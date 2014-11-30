Projects <small> - <a href="https://github.com/landaire">GitHub</a></small>
========
* * *

- [ID3 API](id3/about)
The ID3 API I currently have on my site is meant to be used in conjunction with the [HypeMachine-Extension](https://github.com/fzakaria/HypeMachine-Extension) project by [Farid Zakaria](http://fzakaria.com). It works by utilizing taglib (and [go-taglib](https://github.com/landaire/go-taglib)) to modify ID3v2 tag information of the song passed in the GET request. The current experimental branch of the Hype Machine extension can be found [here](https://github.com/landaire/HypeMachine-Extension/tree/experimental). For a technical rundown on the ID3 API, check out the about page located [here](id3/about).

- [XVal API](xval) (and other implementations)
The XVal API allows anyone to decrypt the x-value displayed on the system info of their Xbox 360 dashboard to see what policy violations their console may have. The JSON API can be used by making a GET request to `/xval.json` with the `serial` and `xval` parameters containg the console serial number and console x-value respectively. The source code can be found [here](https://gist.github.com/landaire/5972627), with [Ruby](https://gist.github.com/landaire/3161073) and [Python](https://gist.github.com/landaire/2669789) implementations also available (note: the Python implementation is the original code by [Redline99](http://www.xboxhacker.org/index.php?topic=16401.0), while the rest are ports done by me).


- [Up](https://github.com/landaire/Up)
A cross-platform FATX (File Allocation Table for Xbox) file system utility. Currently has support for OS X and Windows, Linux (only tested on Ubuntu) and supports reading operations on both Xbox 360-formatted USB drives and hard disks.


- [Party Buffalo](https://code.google.com/p/party-buffalo)
Party Buffalo yet another FATX file system utility I had started developing in August of 2010 when I was 13 years old. It started out as a project to learn more about C# and a simple file system, while at the same time providing a tool for the Xbox 360 community. At the time there was really no decent *free* FATX utility, so I decided to make one. It was the first tool to work for almost any hard drive (other utilities would only work for 20 GB, 60 GB, or 120 GB HDDs) without hardcoding offsets, and was also the first to support reading metadata for various files so users didn't have to blindly figure out what things were on their hard drive. This included reading cached system files when applicable, [STFS](http://free60.org/STFS) metadata, and other types of system files. The project was officially discontinued in 2013, after receiving over 1 million downloads.

- [addpaths.cpp](https://gist.github.com/landaire/3168270)
An app for Xbox 360 development kits which allows users to mount additional system paths so that they can be explored from PCs using Xbox 360 Neighborhood (included in the official Xbox 360 SDK).

- [quote_gabek](https://github.com/landaire/quote_gabek) A Twitter bot to constantly annoy [Gabriel Kirkpatrick](http://twitter.com/gabe_k) by quoting his tweets and adding random things to what he said.

- [music-mover-go](https://github.com/landaire/music-mover-go)
Scans the `~/Downloads` directory for new `.mp3` files, sets the ID3v2 tag information, and moves the file to a target directory.

Open-source projects contributed to: [HypeMachine-Extension](https://github.com/fzakaria/HypeMachine-Extension), [revel](https://github.com/revel/revel)
