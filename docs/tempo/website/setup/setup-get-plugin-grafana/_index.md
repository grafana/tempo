---
title: Set up the GET plugin for Grafana
menuTitle: Set up the GET plugin for Grafana
weight: 350
---

# Set up the GET plugin for Grafana

The Grafana Enterprise Traces (GET) plugin allows you to create new tenants and then use the access page to create new roles to publish traces.
These commands are completed using the Grafana user interface in lieu of commands that would normally be completed using the GET CLI.

This procedure explains how to enable and configure the GET plugin.

## Before you begin

To configure the GET plugin, you need:

* Grafana Enterprise 7.3.0 or higher
* GET cluster access token
* Microservice deployments require the GET gateway URL, for example: `https://gateway.enterprise-traces.svc.cluster.local:3200`

Refer to [Deploy Grafana Enterprise on Kubernetes](https://grafana.com/docs/grafana/latest/installation/kubernetes/#deploy-grafana-enterprise-on-kubernetes) if you are using Kubernetes. Otherwise, refer to [Install Grafana](https://grafana.com/docs/grafana/latest/installation/) for more information.

## Install the plugin in your Grafana Enterprise instance

There are multiple ways to install the plugin to your local Grafana Enterprise instance. 
For more information, refer to [Grafana Enterprise Traces app installation](https://grafana.com/grafana/plugins/grafana-enterprise-traces-app/?tab=installation) (requires login).

After installing the plugin, you will need to restart the Grafana pod for the plugin to be loaded on startup. If using the "Deploy Grafana Enterprise on Kubernetes" instructions, you will need to access the shell for the Grafana pod that you have just deployed to use the GET plugin installation instructions "Installing on a local Grafana".

## Enable and configure the plugin

To enable and configure the plugin:

1. Log in to your Grafana Enterprise.
1. Go to the Config/Plugins page and select the Grafana Enterprise Traces plugin from list. This option is only available if you have GET and Grafana Enterprise licenses installed.
1. From the configuration page of the plugin, enable the plugin by clicking on **Enable plugin**.  
1. Provide the necessary API settings so that the plugin can connect to your cluster:
    1. **Access Token**: Enter the admin-scoped access token that you generated when setting up your GET cluster.
    1. **Enterprise Traces URL**: Enter the URL of your GET cluster. For single-process clusters, this is any node in the cluster. For microservice deployments, this URL is the GET gateway (for example, `http://gateway.enterprise-traces.svc.cluster.local:3200` if you followed the Tanka GET installation instructions).
    1. Choose the relevant API version. For a new installation, select `v3`.
1. Click **Save API settings**.
1. Navigate to Grafana Enterprise Traces (select the GET, the stylised ‘T’ icon, on the menu bar).
1. Verify that the plugin loads and can communicate with the GET admin API endpoints. You can quickly test this by selecting the ‘Licenses’ tab from the GET plugin and ensure that you see something like this:

    ````
    Issuer:
    https://grafana.com
    Issued at:
    2022-06-23 12:37:48
    Expires:
    2023-06-23 12:37:32
    Max number of users:
    0
    Products:
    grafana-enterprise-traces
    ````

1. Look at the default access policy under the **Access Policies** tab and ensure that there is a default access policy for the `admin` scope.

The next step is to [set up a GET tenant and visualize your data]({{< relref "../set-up-get-tenants" >}}).
