.PHONY: clean
clean:
	kubectl delete -f deploy --ignore-not-found
	kubectl delete -f deploy/crds --ignore-not-found

.PHONY: run-local
run-local:
	kubectl apply -f deploy/crds/demo.example.com_etcdclusters_crd.yaml
	operator-sdk run --local --watch-namespace ""

.PHONY: setup
setup:
	kubectl apply -f deploy/crds/demo.example.com_etcdclusters_crd.yaml
	kubectl apply -f deploy/service_account.yaml
	kubectl apply -f deploy/role.yaml
	kubectl apply -f deploy/role_binding.yaml

.PHONY: new-etcd
new-etcd:
	kubectl apply -f deploy/crds/demo.example.com_v1alpha1_etcdcluster_cr.yaml

.PHONY: del-etcd
del-etcd:
	kubectl delete -f deploy/crds/demo.example.com_v1alpha1_etcdcluster_cr.yaml --ignore-not-found

.PHONY: operator-build
operator-build:
ifndef IMAGE_TAG
	@echo IMAGE_TAG not set
	@exit 1
endif
	operator-sdk build ${IMAGE_TAG} --go-build-args "-o build/_output/bin/custom-etcd-operator"
	docker push ${IMAGE_TAG}
	sed -i 's|image:.*|image: '${IMAGE_TAG}'|' deploy/operator.yaml

.PHONY: deploy-operator
deploy-operator: setup
	kubectl apply -f deploy/operator.yaml

.PHONY: olm-install
olm-install:
	operator-sdk olm install

marketplace-install:
	curl -L https://raw.githubusercontent.com/operator-framework/operator-marketplace/master/deploy/upstream/01_namespace.yaml | kubectl apply -f -
	curl -L https://raw.githubusercontent.com/operator-framework/operator-marketplace/master/deploy/upstream/03_operatorsource.crd.yaml | kubectl apply -f -
	curl -L https://raw.githubusercontent.com/operator-framework/operator-marketplace/master/deploy/upstream/04_service_account.yaml | kubectl apply -f -
	curl -L https://raw.githubusercontent.com/operator-framework/operator-marketplace/master/deploy/upstream/05_role.yaml | kubectl apply -f -
	curl -L https://raw.githubusercontent.com/operator-framework/operator-marketplace/master/deploy/upstream/06_role_binding.yaml | kubectl apply -f -
	curl -L https://raw.githubusercontent.com/operator-framework/operator-marketplace/master/deploy/upstream/07_upstream_operatorsource.cr.yaml | kubectl apply -f -
	curl -L https://raw.githubusercontent.com/operator-framework/operator-marketplace/master/deploy/upstream/08_operator.yaml | kubectl apply -f -

.PHONY: hub-setup
hub-setup: olm-install marketplace-install

.PHONY: package-operator
package-operator:
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
ifdef FROM_VERSION
	operator-sdk generate csv --csv-version ${VERSION} --from-version ${FROM_VERSION} --csv-channel default --make-manifests=false --update-crds
else
	operator-sdk generate csv --csv-version ${VERSION} --csv-channel default --make-manifests=false --update-crds
endif

.PHONY: olm-dashboard
olm-dashboard:
	curl -L https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/scripts/run_console_local.sh  | sh

.PHONY: push-quay-app
push-quay-app:
ifndef VERSION
	@echo VERSION not set
	@exit 1
endif
ifndef QUAY_NAMESPACE
	@echo QUAY_NAMESPACE not set
	@exit 1
endif
ifndef TOKEN
	@echo TOKEN not set
	@exit 1
endif
	@operator-courier --verbose push  \
		./deploy/olm-catalog/custom-etcd-operator \
		${QUAY_NAMESPACE} \
		custom-etcd-operator \
		${VERSION}  \
		"${TOKEN}"

.PHONY: olm-clean
olm-clean:
	kubectl delete operatorsource -n marketplace ${QUAY_NAMESPACE}-operators --ignore-not-found

.PHONY: get-quay-token
get-quay-token:
	curl -LO https://raw.githubusercontent.com/operator-framework/operator-courier/master/scripts/get-quay-token
	chmod +x get-quay-token
	./get-quay-token
	rm get-quay-token

#gqt:
#	curl -H "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '
#    {
#        "user": {
#            "username": "'"${USERNAME}"'",
#            "password": "'"${PASSWORD}"'"
#        }
#    }'

export define operatorsource
apiVersion: operators.coreos.com/v1
kind: OperatorSource
metadata:
  name: ${QUAY_NAMESPACE}-operators
  namespace: marketplace
spec:
  type: appregistry
  endpoint: https://quay.io/cnr
  registryNamespace: ${QUAY_NAMESPACE}
  displayName: "${QUAY_NAMESPACE} Operators"
  publisher: "${QUAY_NAMESPACE}"
endef

.PHONY: operator-source
operator-source: olm-clean
	@echo ::::: operator soruce manifest :::::
	@echo "$$operatorsource"
	@echo ::::::::::::::::::::::::::::::
	@echo "$$operatorsource" | kubectl apply -f -

.PHONY: list-operators
list-operators:
	kubectl get opsrc ${QUAY_NAMESPACE}-operators -o=custom-columns=NAME:.metadata.name,PACKAGES:.status.packages -n marketplace


export define operatorgroup
apiVersion: operators.coreos.com/v1alpha2
kind: OperatorGroup
metadata:
  name: my-operatorgroup
  namespace: marketplace
spec:
  targetNamespaces:
  - marketplace
endef

.PHONY: operator-group
operator-group:
	@echo "$$operatorgroup" | kubectl apply -f -

export define subscription
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ${QUAY_NAMESPACE}-custom-etcd-subscription
  namespace: marketplace
spec:
  channel: default
  name: custom-etcd-operator
  source: ${QUAY_NAMESPACE}-operators
  sourceNamespace: marketplace
endef

.PHONY: subscription
subscription:
	@echo ::::: subscription soruce manifest :::::
	@echo "$$subscription"
	@echo ::::::::::::::::::::::::::::::
	@echo "$$subscription" | oc apply -f -