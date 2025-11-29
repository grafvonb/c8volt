//go:build generate

package scripts

//go:generate bash -c "echo 'generating clients...'"

//go:generate bash -c "./1_checkout-latest-tagged-version.sh"
//go:generate bash -c "./2_bundle-camunda-v2-api.sh"
//go:generate bash -c "./3_generate-clients.sh"

//go:generate bash -c "echo 'clients generated'"
