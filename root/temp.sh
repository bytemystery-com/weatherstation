#!/bin/bash
DATESTR=$(date +"%Y-%m-%d")
ROOTDIR='/data/temperature'
mkdir -p "${ROOTDIR}"
LOGFILE="/temperature/temp_${DATESTR}.log"
OUTFILE="${ROOTDIR}/${DATESTR}_temp.png"
GNUPLOTFILE="/root/bin/temp.plot"

if [[ ! -e "${LOGFILE}" ]] ; then
	echo "# Date CPU" > "${LOGFILE}"
	# chown reiner:reiner "${LOGFILE}"
fi

date_str=$(date +'%Y-%m-%d %H:%M:%S')
date2_str=$(date +'%s')
cpu_temp=$(vcgencmd measure_temp | cut -d '=' -f2 | sed 's/[^0-9\.]//g')

echo "${date_str} ${date2_str} ${cpu_temp}" >> "${LOGFILE}"

DSTR="DATE='${DATESTR}'"
FSTR="FILENAME='${LOGFILE}'"
OSTR="OUTFILE='${OUTFILE}'"
OLSTR="OUTFILELOAD='${OUTFILELOAD}'"

gnuplot -e ${DSTR} -e ${FSTR} -e ${OSTR} -e ${OLSTR} "${GNUPLOTFILE}"
# chown reiner:reiner "${OUTFILE}"

