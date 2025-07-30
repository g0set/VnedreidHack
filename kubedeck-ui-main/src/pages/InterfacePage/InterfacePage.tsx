import React, { useEffect, useState } from "react";
import styles from "./InterfacePage.module.css";

// Resource type interfaces based on OpenAPI schema
interface ObjectMeta {
    name: string;
    namespace: string;
    uid: string;
    resourceVersion: string;
    creationTimestamp: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
}

interface ResourceBase {
    kind: string;
    apiVersion: string;
    metadata: ObjectMeta;
}

interface Pod extends ResourceBase {
    status: {
        phase: string;
        podIP: string;
    };
    spec: {
        nodeName: string;
    };
}

interface PersistentVolume extends ResourceBase {
    spec: {
        capacity: {
            storage: string;
        };
        accessModes: string[];
        persistentVolumeReclaimPolicy: string;
        storageClassName: string;
    };
    status: {
        phase: string;
    };
}

interface PersistentVolumeClaim extends ResourceBase {
    spec: {
        accessModes: string[];
        resources: {
            requests: {
                storage: string;
            };
        };
        storageClassName: string;
    };
    status: {
        phase: string;
    };
}

interface ConfigMap extends ResourceBase {
    data?: Record<string, string>;
}

interface Service extends ResourceBase {
    spec: {
        selector: Record<string, string>;
        ports: {
            port: number;
            targetPort: number;
            protocol: string;
        }[];
        type: string;
        clusterIP?: string;
    };
    status: {
        loadBalancer?: {
            ingress?: {
                ip: string;
            }[];
        };
    };
}

interface Deployment extends ResourceBase {
    spec: {
        replicas: number;
        selector: {
            matchLabels: Record<string, string>;
        };
    };
    status: {
        replicas: number;
        availableReplicas: number;
        readyReplicas?: number;
        updatedReplicas?: number;
    };
}

interface Namespace extends ResourceBase {
    spec: {
        finalizers: string[];
    };
    status: {
        phase: string;
    };
}

type ResourceType =
    | "pods"
    | "pv"
    | "pvc"
    | "configmaps"
    | "services"
    | "ingresses"
    | "deployments"
    | "statefulsets"
    | "daemonsets"
    | "replicasets"
    | "nodes"
    | "namespaces"
    | "storageclasses";

const API_BASE_URL = "https://ui.orion.nikcorp.ru/operator";

export const InterfacePage: React.FC = () => {
    const [activeResource, setActiveResource] = useState<ResourceType>("pods");
    const [selectedNamespace, setSelectedNamespace] = useState<string>("");
    const [namespaces, setNamespaces] = useState<string[]>([]);
    const [resources, setResources] = useState<ResourceBase[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

    const fetchNamespaces = async () => {
        try {
            const response = await fetch(`${API_BASE_URL}/namespaces`);
            const data = await response.json();

            if (Array.isArray(data)) {
                setNamespaces(data.map((ns: Namespace) => ns.metadata.name));
            } else {
                setNamespaces([]);
            }
        } catch (err) {
            setError("Failed to fetch namespaces");
            console.error("Error fetching namespaces:", err);
        }
    };

    const fetchResources = async (
        resourceType: ResourceType,
        namespace?: string,
    ) => {
        setLoading(true);
        setError(null);
        try {
            const url = namespace
                ? `${API_BASE_URL}/${resourceType}?namespace=${namespace}`
                : `${API_BASE_URL}/${resourceType}`;
            const response = await fetch(url);
            const data = await response.json();

            if (Array.isArray(data)) {
                setResources(data);
            } else {
                setResources([]);
            }
        } catch (err) {
            setError(`Failed to fetch ${resourceType}`);
            console.error(`Error fetching ${resourceType}:`, err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchNamespaces();
    }, []);

    useEffect(() => {
        fetchResources(activeResource, selectedNamespace);
    }, [activeResource, selectedNamespace]);

    // Helper functions for UI rendering
    const getStatusClass = (status: string): string => {
        switch (status.toLowerCase()) {
            case "running":
            case "bound":
            case "active":
            case "available":
                return styles.statusRunning;
            case "pending":
            case "terminating":
                return styles.statusPending;
            case "failed":
            case "error":
                return styles.statusFailed;
            default:
                return "";
        }
    };

    const formatDate = (dateString: string): string => {
        return new Date(dateString).toLocaleString();
    };

    // Render resource-specific table
    const renderResourceTable = () => {
        if (loading) {
            return <div className={styles.loading}>Loading...</div>;
        }

        if (resources.length === 0) {
            return <div className={styles.empty}>No resources found</div>;
        }

        switch (activeResource) {
            case "pods":
                return (
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th className={styles.tableHeader}>Name</th>
                                <th className={styles.tableHeader}>
                                    Namespace
                                </th>
                                <th className={styles.tableHeader}>Status</th>
                                <th className={styles.tableHeader}>IP</th>
                                <th className={styles.tableHeader}>Node</th>
                                <th className={styles.tableHeader}>Created</th>
                            </tr>
                        </thead>
                        <tbody>
                            {resources.map((resource) => {
                                const pod = resource as Pod;
                                return (
                                    <tr
                                        key={`${pod.metadata.namespace}/${pod.metadata.name}`}
                                        className={styles.tableRow}
                                    >
                                        <td className={styles.tableCell}>
                                            {pod.metadata.name}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pod.metadata.namespace}
                                        </td>
                                        <td className={styles.tableCell}>
                                            <span
                                                className={`${styles.status} ${getStatusClass(pod.status.phase)}`}
                                            >
                                                {pod.status.phase}
                                            </span>
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pod.status.podIP}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pod.spec?.nodeName || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {formatDate(
                                                pod.metadata.creationTimestamp,
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                );

            case "pv":
                return (
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th className={styles.tableHeader}>Name</th>
                                <th className={styles.tableHeader}>Status</th>
                                <th className={styles.tableHeader}>Capacity</th>
                                <th className={styles.tableHeader}>
                                    Access Modes
                                </th>
                                <th className={styles.tableHeader}>
                                    Storage Class
                                </th>
                                <th className={styles.tableHeader}>Created</th>
                            </tr>
                        </thead>
                        <tbody>
                            {resources.map((resource) => {
                                const pv = resource as PersistentVolume;
                                return (
                                    <tr
                                        key={pv.metadata.name}
                                        className={styles.tableRow}
                                    >
                                        <td className={styles.tableCell}>
                                            {pv.metadata.name}
                                        </td>
                                        <td className={styles.tableCell}>
                                            <span
                                                className={`${styles.status} ${getStatusClass(pv.status.phase)}`}
                                            >
                                                {pv.status.phase}
                                            </span>
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pv.spec.capacity?.storage || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pv.spec.accessModes?.join(", ") ||
                                                "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pv.spec.storageClassName || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {formatDate(
                                                pv.metadata.creationTimestamp,
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                );

            case "pvc":
                return (
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th className={styles.tableHeader}>Name</th>
                                <th className={styles.tableHeader}>
                                    Namespace
                                </th>
                                <th className={styles.tableHeader}>Status</th>
                                <th className={styles.tableHeader}>Capacity</th>
                                <th className={styles.tableHeader}>
                                    Access Modes
                                </th>
                                <th className={styles.tableHeader}>
                                    Storage Class
                                </th>
                                <th className={styles.tableHeader}>Created</th>
                            </tr>
                        </thead>
                        <tbody>
                            {resources.map((resource) => {
                                const pvc = resource as PersistentVolumeClaim;
                                return (
                                    <tr
                                        key={`${pvc.metadata.namespace}/${pvc.metadata.name}`}
                                        className={styles.tableRow}
                                    >
                                        <td className={styles.tableCell}>
                                            {pvc.metadata.name}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pvc.metadata.namespace}
                                        </td>
                                        <td className={styles.tableCell}>
                                            <span
                                                className={`${styles.status} ${getStatusClass(pvc.status.phase)}`}
                                            >
                                                {pvc.status.phase}
                                            </span>
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pvc.spec.resources?.requests
                                                ?.storage || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pvc.spec.accessModes?.join(", ") ||
                                                "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {pvc.spec.storageClassName || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {formatDate(
                                                pvc.metadata.creationTimestamp,
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                );

            case "configmaps":
                return (
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th className={styles.tableHeader}>Name</th>
                                <th className={styles.tableHeader}>
                                    Namespace
                                </th>
                                <th className={styles.tableHeader}>
                                    Data Keys
                                </th>
                                <th className={styles.tableHeader}>Created</th>
                            </tr>
                        </thead>
                        <tbody>
                            {resources.map((resource) => {
                                const cm = resource as ConfigMap;
                                return (
                                    <tr
                                        key={`${cm.metadata.namespace}/${cm.metadata.name}`}
                                        className={styles.tableRow}
                                    >
                                        <td className={styles.tableCell}>
                                            {cm.metadata.name}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {cm.metadata.namespace}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {cm.data
                                                ? Object.keys(cm.data).join(
                                                      ", ",
                                                  )
                                                : "No data"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {formatDate(
                                                cm.metadata.creationTimestamp,
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                );

            case "services":
                return (
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th className={styles.tableHeader}>Name</th>
                                <th className={styles.tableHeader}>
                                    Namespace
                                </th>
                                <th className={styles.tableHeader}>Type</th>
                                <th className={styles.tableHeader}>
                                    Cluster IP
                                </th>
                                <th className={styles.tableHeader}>
                                    External IP
                                </th>
                                <th className={styles.tableHeader}>Ports</th>
                                <th className={styles.tableHeader}>Created</th>
                            </tr>
                        </thead>
                        <tbody>
                            {resources.map((resource) => {
                                const svc = resource as Service;
                                return (
                                    <tr
                                        key={`${svc.metadata.namespace}/${svc.metadata.name}`}
                                        className={styles.tableRow}
                                    >
                                        <td className={styles.tableCell}>
                                            {svc.metadata.name}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {svc.metadata.namespace}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {svc.spec.type}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {svc.spec.clusterIP || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {svc.status?.loadBalancer?.ingress
                                                ?.map((ing) => ing.ip)
                                                .join(", ") || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {svc.spec.ports
                                                ?.map(
                                                    (p) =>
                                                        `${p.port}:${p.targetPort}/${p.protocol}`,
                                                )
                                                .join(", ") || "N/A"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {formatDate(
                                                svc.metadata.creationTimestamp,
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                );

            case "deployments":
                return (
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th className={styles.tableHeader}>Name</th>
                                <th className={styles.tableHeader}>
                                    Namespace
                                </th>
                                <th className={styles.tableHeader}>Ready</th>
                                <th className={styles.tableHeader}>
                                    Up-to-date
                                </th>
                                <th className={styles.tableHeader}>
                                    Available
                                </th>
                                <th className={styles.tableHeader}>Age</th>
                            </tr>
                        </thead>
                        <tbody>
                            {resources.map((resource) => {
                                const deploy = resource as Deployment;
                                return (
                                    <tr
                                        key={`${deploy.metadata.namespace}/${deploy.metadata.name}`}
                                        className={styles.tableRow}
                                    >
                                        <td className={styles.tableCell}>
                                            {deploy.metadata.name}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {deploy.metadata.namespace}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {deploy.status.readyReplicas || 0}/
                                            {deploy.status.replicas}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {deploy.status.updatedReplicas || 0}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {deploy.status.availableReplicas ||
                                                0}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {formatDate(
                                                deploy.metadata
                                                    .creationTimestamp,
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                );

            case "namespaces":
                return (
                    <table className={styles.table}>
                        <thead>
                            <tr>
                                <th className={styles.tableHeader}>Name</th>
                                <th className={styles.tableHeader}>Status</th>
                                <th className={styles.tableHeader}>Labels</th>
                                <th className={styles.tableHeader}>Created</th>
                            </tr>
                        </thead>
                        <tbody>
                            {resources.map((resource) => {
                                const ns = resource as Namespace;
                                return (
                                    <tr
                                        key={ns.metadata.name}
                                        className={styles.tableRow}
                                    >
                                        <td className={styles.tableCell}>
                                            {ns.metadata.name}
                                        </td>
                                        <td className={styles.tableCell}>
                                            <span
                                                className={`${styles.status} ${getStatusClass(ns.status.phase)}`}
                                            >
                                                {ns.status.phase}
                                            </span>
                                        </td>
                                        <td className={styles.tableCell}>
                                            {ns.metadata.labels
                                                ? Object.entries(
                                                      ns.metadata.labels,
                                                  )
                                                      .map(
                                                          ([k, v]) =>
                                                              `${k}: ${v}`,
                                                      )
                                                      .join(", ")
                                                : "No labels"}
                                        </td>
                                        <td className={styles.tableCell}>
                                            {formatDate(
                                                ns.metadata.creationTimestamp,
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                );

            // Additional resource types could be implemented here

            default:
                return (
                    <div className={styles.empty}>
                        Select a resource type to view
                    </div>
                );
        }
    };

    return (
        <div className={styles.container}>
            <h1 className={styles.title}>Kubernetes Resources</h1>

            <div className={styles.tabs}>
                <button
                    className={`${styles.tabButton} ${activeResource === "pods" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("pods")}
                >
                    Pods
                </button>
                <button
                    className={`${styles.tabButton} ${activeResource === "deployments" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("deployments")}
                >
                    Deployments
                </button>
                <button
                    className={`${styles.tabButton} ${activeResource === "services" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("services")}
                >
                    Services
                </button>
                <button
                    className={`${styles.tabButton} ${activeResource === "configmaps" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("configmaps")}
                >
                    ConfigMaps
                </button>
                <button
                    className={`${styles.tabButton} ${activeResource === "ingresses" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("ingresses")}
                >
                    Ingresses
                </button>
                <button
                    className={`${styles.tabButton} ${activeResource === "pv" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("pv")}
                >
                    PersistentVolumes
                </button>
                <button
                    className={`${styles.tabButton} ${activeResource === "pvc" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("pvc")}
                >
                    PersistentVolumeClaims
                </button>
                <button
                    className={`${styles.tabButton} ${activeResource === "namespaces" ? styles.activeTab : ""}`}
                    onClick={() => setActiveResource("namespaces")}
                >
                    Namespaces
                </button>
            </div>

            <div className={styles.resourceSection}>
                <div className={styles.resourceHeader}>
                    <h2 className={styles.resourceTitle}>
                        {activeResource.charAt(0).toUpperCase() +
                            activeResource.slice(1)}
                    </h2>
                    <div className={styles.controls}>
                        {activeResource !== "namespaces" &&
                            activeResource !== "nodes" &&
                            activeResource !== "pv" &&
                            activeResource !== "storageclasses" && (
                                <select
                                    className={styles.select}
                                    value={selectedNamespace}
                                    onChange={(e) =>
                                        setSelectedNamespace(e.target.value)
                                    }
                                >
                                    <option value="">All Namespaces</option>
                                    {namespaces.map((ns) => (
                                        <option key={ns} value={ns}>
                                            {ns}
                                        </option>
                                    ))}
                                </select>
                            )}
                        <button
                            className={styles.refreshButton}
                            onClick={() =>
                                fetchResources(
                                    activeResource,
                                    selectedNamespace,
                                )
                            }
                        >
                            Refresh
                        </button>
                    </div>
                </div>

                {error && <div className={styles.error}>{error}</div>}

                {renderResourceTable()}
            </div>
        </div>
    );
};
