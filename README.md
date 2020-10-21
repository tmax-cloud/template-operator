# Template-operator

> Template-operator for HyperCloud Service

**Architecture**

## prerequisite Install
- kubernetes

## Install Template Operator

- [CRD](#crd)
- [Namespace](#namespace)
- [ServiceAccount](#serviceaccount)
- [Role](#role)
- [RoleBinding](#rolebinding)
- [Deployment](#deployment)
- [Test](#test)

---

#### crd
> Apply crd
- kubectl apply -f tmax.io_templates_crd.yaml ([파일](./deploy/crds/tmax.io_templates_crd.yaml))
- kubectl apply -f tmax.io_clustertemplates_crd.yaml ([파일](./deploy/crds/tmax.io_clustertemplates_crd.yaml))
- kubectl apply -f tmax.io_templateinstances_crd.yaml ([파일](./deploy/crds/tmax.io_templateinstances_crd.yaml))
- kubectl apply -f tmax.io_catalogserviceclaims_crd.yaml ([파일](./deploy/crds/tmax.io_catalogserviceclaims_crd.yaml))

---

#### Namespace
> Create your own namespace.
- kubectl create namespace {YOUR_NAMESPACE}

---

#### ServiceAccount
> Create a service account to pass permission to template operator.

- kubectl apply -f service_account.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/service_account.yaml))

---

#### Role
> Create a role.

- kubectl apply -f role.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/role.yaml))

---

#### RoleBinding
> Create a RoleBinding. Bind Role and Service Account.
>> You should change namespace to {YOUR_NAMESPACE} in role_biding.yaml file

- kubectl apply -f role_binding.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/role_binding.yaml))

---

#### Deployment
> Create Template Operator.

- kubectl apply -f operator.yaml -n {YOUR_NAMESPACE} ([파일](./deploy/operator.yaml))

---

#### Test
> test yaml in example directory.

- kubectl apply -f example.yaml

---