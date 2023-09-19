#!/bin/sh

check_number() {
  local number="$1"

  if [ -z "$number" ]; then
    echo "\"$number\" is not a valid number"
    exit 1
  fi

  re='^[0-9]+$'
  if [[ $number =~ $re ]] ; then
    echo
  else
    echo "\"$number\" is not a valid number"
    exit 1
  fi
}

finish() {
  code=$?
  curl -s -XPOST http://127.0.0.1:15020/quitquitquit || true
  exit $code
}

trap finish EXIT

[[ -z "$NAMESPACE" ]] && { echo "NAMESPACE is required" ; exit 1; }

[[ -z "$WORKLOADS" ]] && { echo "WORKLOADS is required" ; exit 1; }

if [ -z "$WAIT_SECONDS" ]; then
  WAIT_SECONDS="360"
fi

next_version=""
if [ -f /opt/versions/version ]; then
  next_version=$(cat /opt/versions/version)
fi
echo "next_version: $next_version"

if [ -z "$next_version" ]; then
  echo "failed to get next version from helm chart"
  exit 1
fi

next_major_version=$(echo "$next_version" | awk -F'.' '{print $1}')
check_number "$next_major_version"

updated_workloads=""

export IFS=";"
for workload in $WORKLOADS; do
  name=$(echo $workload | awk -F',' '{print $1}')
  type=$(echo $workload | awk -F',' '{print $2}')
  container=$(echo $workload | awk -F',' '{print $3}')

  current_version=$(kubectl -n $NAMESPACE get $type $name  -o json | jq -r --arg container "$container" '.spec.template.spec.containers[] | select(.name==$container) | .image' | awk -F':' '{print $2}')
  echo "current_version: $current_version"

  if [ -z "$current_version" ]; then
    echo "failed to get current version of workload $type/$name"
    exit 1
  fi

  current_major_version=$(echo "$current_version" | awk -F'.' '{print $1}')
  check_number "$current_major_version"

  if [ $current_major_version != $next_major_version ]; then
    updated_workloads="${updated_workloads} ${type}/${name}"
  fi
done

if [ -n "$updated_workloads" ]; then
  kubectl -n $NAMESPACE scale $updated_workloads --replicas=0
  date
  sleep $WAIT_SECONDS
fi
