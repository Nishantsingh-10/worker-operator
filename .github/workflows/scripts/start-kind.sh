#!/bin/bash

# Create controller kind cluster if not present
if [ ! $(kind get clusters | grep controller) ];then
  kind create cluster --name controller --config .github/workflows/scripts/cluster.yaml --image kindest/node:v1.22.7

  function wait_for_pods {
  for ns in "$namespace"; do
    for pod in $(kubectl get pods -n $ns | grep -v NAME | awk '{ print $1 }'); do
      counter=0
      echo kubectl get pod $pod -n $ns
      kubectl get pod $pod -n $ns
      while [[ ! ($(kubectl get po $pod -n $ns | grep $pod | awk '{print $3}') =~ ^Running$|^Completed$) ]]; do
        sleep 1
        let counter=counter+1

        if ((counter == $sleep)); then
          echo "POD $pod failed to start in $sleep seconds"
          kubectl get events -n $ns --sort-by='.lastTimestamp'
          echo "Exiting"

          exit -1
        fi
      done
    done
  done
}

snap install kubectx --classic

# Install Calico in Controller...
echo Switch to controller context and Install Calico...
kubectx kind-controller
kubectx

echo Install the Tigera Calico operator...
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/tigera-operator.yaml

echo "Check for tigera-operator pods"
kubectl get pods -n tigera-operator
echo "Wait for tigera-operator to be Running"
namespace=tigera-operator
sleep=60
wait_for_pods

kubectl get pods -n tigera-operator

echo Install the custom resource definitions manifest...
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/custom-resources.yaml
sleep 10

echo "Check for calico-system pods"
kubectl get pods -n calico-system
echo "Wait for Calico-system to be Running"
namespace=calico-system
sleep=600
wait_for_pods

kubectl get pods -n calico-system

sleep 10
echo "Check for calico-apiserver pods"
kubectl get pods -n calico-apiserver
echo "Wait for calico-apiserver to be Running"
namespace=calico-apiserver
sleep=120
wait_for_pods

kubectl get pods -n calico-apiserver


  ip=$(docker inspect controller-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress') 
#  echo $ip
# loading docker image into kind controller
  kind load docker-image worker-operator:e2e-latest
# Replace loopback IP with docker ip
  kind get kubeconfig --name controller | sed "s/127.0.0.1.*/$ip:6443/g" > /home/runner/.kube/kind1.yaml
fi


# Create worker1 kind cluster if not present

if [ ! $(kind get clusters | grep worker) ];then
  kind create cluster --name worker --config .github/workflows/scripts/cluster.yaml --image kindest/node:v1.22.7
  
  function wait_for_pods {
  for ns in "$namespace"; do
    for pod in $(kubectl get pods -n $ns | grep -v NAME | awk '{ print $1 }'); do
      counter=0
      echo kubectl get pod $pod -n $ns
      kubectl get pod $pod -n $ns
      while [[ ! ($(kubectl get po $pod -n $ns | grep $pod | awk '{print $3}') =~ ^Running$|^Completed$) ]]; do
        sleep 1
        let counter=counter+1

        if ((counter == $sleep)); then
          echo "POD $pod failed to start in $sleep seconds"
          kubectl get events -n $ns --sort-by='.lastTimestamp'
          echo "Exiting"

          exit -1
        fi
      done
    done
  done
}

snap install kubectx --classic

# Install Calico in Controller...
echo Switch to controller context and Install Calico...
kubectx kind-worker
kubectx

echo Install the Tigera Calico operator...
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/tigera-operator.yaml

echo "Check for tigera-operator pods"
kubectl get pods -n tigera-operator
echo "Wait for tigera-operator to be Running"
namespace=tigera-operator
sleep=60
wait_for_pods

kubectl get pods -n tigera-operator


echo Install the custom resource definitions manifest...
kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.24.1/manifests/custom-resources.yaml
sleep 10

echo "Check for calico-system pods"
kubectl get pods -n calico-system
echo "Wait for Calico-system to be Running"
namespace=calico-system
sleep=600
wait_for_pods

kubectl get pods -n calico-system

sleep 10
echo "Check for calico-apiserver pods"
kubectl get pods -n calico-apiserver
echo "Wait for calico-apiserver to be Running"
namespace=calico-apiserver
sleep=120
wait_for_pods

kubectl get pods -n calico-apiserver

  ip=$(docker inspect worker-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress')
#  echo $ip
# loading docker image into kind worker
   kind load docker-image worker-operator:e2e-latest

# Replace loopback IP with docker ip
  kind get kubeconfig --name worker | sed "s/127.0.0.1.*/$ip:6443/g" > /home/runner/.kube/kind2.yaml
fi

KUBECONFIG=/home/runner/.kube/kind1.yaml:/home/runner/.kube/kind2.yaml kubectl config view --raw  > /home/runner/.kube/kinde2e.yaml

if [ ! -f profile/kind.yaml ];then
  # Provide correct IP in kind profile, since worker operator cannot detect internal IP as nodeIp
  IP1=$(docker inspect controller-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress')
  IP2=$(docker inspect worker-control-plane | jq -r '.[0].NetworkSettings.Networks.kind.IPAddress')

  cat > profile/kind.yaml << EOF
Kubeconfig: kinde2e.yaml
ControllerCluster:
  Context: kind-controller
  HubChartOptions:
      Repo: "https://kubeslice.github.io/kubeslice/"
WorkerClusters:
- Context: kind-controller
  NodeIP: ${IP1}
- Context: kind-worker
  NodeIP: ${IP2}
WorkerChartOptions:
  Repo: https://kubeslice.github.io/kubeslice/
  SetStrValues:
    "operator.image": "worker-operator"
    "operator.tag": "e2e-latest"
TestSuitesEnabled:
  HubSuite: false
  WorkerSuite: true
EOF

fi
