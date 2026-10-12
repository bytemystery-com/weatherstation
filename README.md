# Weather station

![alt text](/websever/web/favicon.svg "Logo")

Weather station is my private project for monitoring temperature, humidity, air pressure and ambient light.
I originally developed the system using an ESP32. But ESP IDF changes frequently, often breaking compatibility with existing code so I decided to rewrite the projekt based on Raspberry Pi and [Go](https://go.dev/).
I use a Raspberry Pi 5 with 1 GB of RAM and a 64 GByte Lexar SD-card.  
As sensors I use:

- SHT45 for temperature and humidity  
- BMP581 for pressure  
- VEML7700 for ambient light

The sensors are installed in a sensor enclosure connected to the Raspberry Pi via a 6 m cable using I²C communication. I have added a LTC4311 to improve signal quality - but tests have shown that it also
works without it.

Data are filtered with a two Exponential Moving Average (EMA) filters with different filter constants.
So we get short-term and long-term filtered data.  
For pressure we also calculate a one-hour and three-hour trends.

Data are written to SD card in csv format.
Actual and historical data are displayed on a website.

Author: Reiner Pröls  
License: MIT  

## Screenshots

![alt text](/screenshots/sensorbox.jpg "Sensorbox")
![alt text](/screenshots/raspberry.jpg "Raspberry Pi")
![alt text](/screenshots/web_main.jpg "Website main page")
![alt text](/screenshots/web_config.jpg "Website config")

## Links

- [Readme](https://bytemystery-com.github.io/weatherstation/)
- [Repository](https://github.com/bytemystery-com/weatherstation/)

© Copyright Reiner Pröls, 2026-2027

