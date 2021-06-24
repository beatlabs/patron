#!/bin/bash

# directory of this script
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# temporary directories
tmp_folder="${script_dir}"/tmp
bin_folder="${tmp_folder}"/bin
log_folder="${tmp_folder}"/log
pid_file="${tmp_folder}"/pid.txt

# assets folder (used only by http-svc)
export PATRON_EXAMPLE_ASSETS_FOLDER="${script_dir}"/http/public

# build the svc binary into convention folder
function build_bin {
  local svc_name=$1
  local src_folder="${script_dir}/$2"
  go build -o "${bin_folder}/${svc_name}" "${src_folder}"/main.go
}

# starts svc redirecting stdout/stderr to file and adding pid to file
function start_svc {
  local svc_name=$1
  "${bin_folder}/${svc_name}" >> "${log_folder}/${svc_name}.log" 2>&1 &
  echo "${!}|${svc_name}" >> "${pid_file}"
}

# kill currently running services relying on the pid file
function stop_running_processes {
  if test -f "${pid_file}"; then
    while read -r line
    do
      pid=$(echo "${line}" | cut -d '|' -f1);
      svc=$(echo "${line}" | cut -d '|' -f2);
      echo "killing service ${svc} with pid ${pid}"
      kill "${pid}"
    done < "${pid_file}"
    rm "${pid_file}"
  fi
}

action=$1

# if action is stop only stops processes currently running
if [ "${action}" == "stop" ]; then
  stop_running_processes
  exit $?
elif [ "${action}" == "clean" ]; then
  stop_running_processes
  rm -fr ${tmp_folder}
  exit $?
elif [ "${action}" == "start" ]; then
  stop_running_processes
  # clean everything
  rm -fr "${bin_folder}"
  mkdir -p "${bin_folder}"
  mkdir -p "${log_folder}"

  # http cache service (http-cache-svc)
  build_bin http-cache-svc http-cache
  start_svc http-cache-svc

  # http service (http-svc)
  build_bin http-svc http
  start_svc http-svc

  # http service (http-sec-svc)
  build_bin http-sec-svc http-sec
  start_svc http-sec-svc

  # kafka consumer (http-kafka-svc)
  build_bin http-kafka-svc kafka
  start_svc http-kafka-svc

  # amqp consumer (http-amqp-svc)
  build_bin http-amqp-svc amqp
  start_svc http-amqp-svc

  # sqs consumer (http-sqs-svc)
  build_bin http-sqs-svc sqs
  start_svc http-sqs-svc

  # grpc service (http-grpc-svc)
  build_bin http-grpc-svc grpc
  start_svc http-grpc-svc
else
  echo "Usage: $0 [stop|clean|start]"
fi
