# set terminal x11 font "Helvetica,20"
set terminal png font ",16" size 1920,1280
set output OUTFILE
set title "Temperature for ".DATE
set xlabel "Time"
set grid
set key left
set style line 1 lt rgb "#ff0000" lw 3 pt 7 ps 1.0
set datafile separator " "
set ytics 2
# set ytics 10
# set mytics 5
set ylabel 'Temperature [°C]'
set xdata time
set format x '%H'
set timefmt '%Y-%m-%d %H:%M:%S'
set xtics 1 hour

plot FILENAME using 1:4 axes x1y1 title 'CPU' with linespoints ls 1
