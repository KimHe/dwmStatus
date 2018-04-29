dwmStatus
===============
![](https://github.com/KimHe/dwmStatus/blob/master/demo.png)

# Introduction
This is a program for customizing your status bar when you are using dwm windows manager.
It was originally written by "ergio correia" in go language; I changed the display style and added temperature information.
I feel someone like me might be also interested in informative dwm status bar, so I distributed the source code here.

# Available information
    - upload and download internet
    - brightness 
    - temperature
    - battery
    - CPU
    - memory
    - date
    - time

if your need one or some of the above information be displayed in your status bar, please add entries into the json file.

# How to build
    requirements:

    * go compiler
    * dwm windows manager
    * awesome font

    build:
    (sudo) make dwmStatus install

# Others
The network interface differs in your case, you need change wlp3s0.
If you do not have idea, please use "ifconfig" to check.

The program should be run like
    dwmStatus dwmStatus.json
by passing json file as parameters

Lastly, you should put the above command where you start dwm. 

