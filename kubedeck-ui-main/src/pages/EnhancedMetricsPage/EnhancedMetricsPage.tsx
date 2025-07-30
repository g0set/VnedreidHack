import React, { useEffect, useState, useRef } from "react";
import CpuChart from "../../coms/grafics/CpuChart";
import styles from "./EnhancedMetricsPage.module.css";

type PodInfo = {
    name: string;
    resourceType: string;
    resourceName: string;
    cpuRequest: number;
    memoryRequest: number;
    cpuLimit: number;
    memoryLimit: number;
    status: string;
    replicas: number;
};

type PodValues = {
    cpuLimit?: number;
    cpuRequest?: number;
    memoryLimit?: number;
    memoryRequest?: number;
    replicas?: number;
};

type ApiData = {
    message: string;
    namespaces: {
        [namespace: string]: PodInfo[];
    };
};

interface DataToFetch {
    namespace: string;
    resource: {
        resourceType: string;
        name: string;
        cpuRequest: number;
        memoryRequest: number;
        cpuLimit: number;
        memoryLimit: number;
        replicas: number;
    };
}

export const EnhancedMetricsPage: React.FC = () => {
    const [data, setData] = useState<ApiData | null>(null);
    const [podValues, setPodValues] = useState<Record<string, PodValues>>({});
    const [cpuIntervalMs, setCpuIntervalMs] = useState(5 * 60 * 1000);
    const [memIntervalMs, setMemIntervalMs] = useState(5 * 60 * 1000);
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [fetchingRecommendations, setFetchingRecommendations] =
        useState<boolean>(true);
    const [activeTooltip, setActiveTooltip] = useState<string | null>(null);
    const [successMessage, setSuccessMessage] = useState<string | null>(null);
    const timeoutRef = useRef<number | null>(null);

    useEffect(() => {
        fetchRecommendations();
    }, []);

    const fetchRecommendations = async () => {
        setFetchingRecommendations(true);
        try {
            const response = await fetch(
                "https://ui.orion.nikcorp.ru/operator/analyze/resources",
            );
            const json: ApiData = await response.json();
            setData(json);

            // Initialize form values from fetched data
            const initialValues: Record<string, PodValues> = {};
            for (const namespace in json.namespaces) {
                for (const pod of json.namespaces[namespace]) {
                    initialValues[pod.name] = {
                        cpuLimit: pod.cpuLimit,
                        cpuRequest: pod.cpuRequest,
                        memoryLimit: pod.memoryLimit,
                        memoryRequest: pod.memoryRequest,
                        replicas: pod.replicas,
                    };
                }
            }
            setPodValues(initialValues);
        } catch (err) {
            console.error("Error loading metrics:", err);
        } finally {
            setFetchingRecommendations(false);
        }
    };

    const handleChange = (
        podName: string,
        field: keyof PodValues,
        value: string,
    ) => {
        setPodValues((prev) => ({
            ...prev,
            [podName]: {
                ...prev[podName],
                [field]: Number(value),
            },
        }));
    };

    const applyChanges = async (namespace: string, pod: PodInfo) => {
        const values = podValues[pod.name] || {};
        const dataToFetch: DataToFetch = {
            namespace: namespace,
            resource: {
                resourceType: pod.resourceType,
                name: pod.resourceName,
                cpuRequest: values.cpuRequest ?? 0,
                memoryRequest: values.memoryRequest ?? 0,
                cpuLimit: values.cpuLimit ?? 0,
                memoryLimit: values.memoryLimit ?? 0,
                replicas: values.replicas ?? 0,
            },
        };

        try {
            setIsLoading(true);
            const response = await fetch(
                "https://ui.orion.nikcorp.ru/operator/updatefromfront",
                {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify(dataToFetch),
                },
            );

            if (!response.ok) {
                throw new Error("Failed to update resources");
            }

            // Show success message
            setSuccessMessage(`Resources for ${pod.name} successfully updated`);

            // Clear success message immediately after applying changes
            setSuccessMessage(null);
        } catch (error) {
            console.error("Error updating resources:", error);
            // Error handling could be added here
        } finally {
            setIsLoading(false);
        }
    };

    // Clear timeout on unmount
    useEffect(() => {
        return () => {
            if (timeoutRef.current) {
                window.clearTimeout(timeoutRef.current);
            }
        };
    }, []);

    // Show tooltip for resource input field
    const showTooltip = (key: string) => {
        setActiveTooltip(key);
    };

    // Hide tooltip
    const hideTooltip = () => {
        setActiveTooltip(null);
    };

    // Format resource type name for better readability
    const formatResourceType = (type: string): string => {
        switch (type.toLowerCase()) {
            case "deployment":
                return "Deployment";
            case "statefulset":
                return "StatefulSet";
            case "daemonset":
                return "DaemonSet";
            case "pod":
                return "Pod";
            default:
                return type.charAt(0).toUpperCase() + type.slice(1);
        }
    };

    return (
        <div className={styles.container}>
            {successMessage && (
                <div className={styles.successNotification}>
                    <svg
                        width="20"
                        height="20"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                    >
                        <path
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            strokeWidth={2}
                            d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                        />
                    </svg>
                    {successMessage}
                    <button
                        onClick={() => setSuccessMessage(null)}
                        className={styles.closeButton}
                    >
                        ×
                    </button>
                </div>
            )}
            <h1 className={styles.title}>Kubernetes Metrics Dashboard</h1>

            <h2 className={styles.subtitle}>
                <svg
                    width="24"
                    height="24"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                >
                    <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M13 10V3L4 14h7v7l9-11h-7z"
                    />
                </svg>
                Рекомендации по оптимизации ресурсов
            </h2>

            {fetchingRecommendations ? (
                <div className={styles.loadingContainer}>
                    <div className={styles.loadingSpinner} />
                    <p className={styles.loadingText}>
                        Generating optimization recommendations...
                    </p>
                </div>
            ) : (
                data && (
                    <div className={styles.recommendationsSection}>
                        <div className={styles.recommendationsHeader}>
                            <h3 className={styles.recommendationsTitle}>
                                <svg
                                    width="20"
                                    height="20"
                                    fill="none"
                                    viewBox="0 0 24 24"
                                    stroke="currentColor"
                                >
                                    <path
                                        strokeLinecap="round"
                                        strokeLinejoin="round"
                                        strokeWidth={2}
                                        d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                                    />
                                </svg>
                                Системные Рекомендации
                            </h3>
                            <button
                                className={styles.refreshButton}
                                onClick={fetchRecommendations}
                                disabled={fetchingRecommendations}
                            >
                                <svg
                                    width="16"
                                    height="16"
                                    fill="none"
                                    viewBox="0 0 24 24"
                                    stroke="currentColor"
                                >
                                    <path
                                        strokeLinecap="round"
                                        strokeLinejoin="round"
                                        strokeWidth={2}
                                        d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                                    />
                                </svg>
                                {fetchingRecommendations
                                    ? "Обновление..."
                                    : "Обновить рекомендации"}
                            </button>
                        </div>
                        <p className={styles.recommendationsDescription}>
                            {data.message}
                        </p>

                        {Object.entries(data.namespaces).map(
                            ([namespace, pods]) => (
                                <div
                                    key={namespace}
                                    className={styles.namespaceSection}
                                >
                                    <h4 className={styles.namespaceTitle}>
                                        <svg
                                            width="16"
                                            height="16"
                                            fill="none"
                                            viewBox="0 0 24 24"
                                            stroke="currentColor"
                                        >
                                            <path
                                                strokeLinecap="round"
                                                strokeLinejoin="round"
                                                strokeWidth={2}
                                                d="M5 19a2 2 0 01-2-2V7a2 2 0 012-2h4l2 2h4a2 2 0 012 2v1M5 19h14a2 2 0 002-2v-5a2 2 0 00-2-2H9a2 2 0 00-2 2v5a2 2 0 01-2 2z"
                                            />
                                        </svg>
                                        Namespace: {namespace}
                                    </h4>
                                    <div className={styles.podsList}>
                                        {pods.map((pod) => {
                                            const values =
                                                podValues[pod.name] || {};
                                            return (
                                                <div
                                                    key={pod.name}
                                                    className={styles.podCard}
                                                >
                                                    <div
                                                        className={
                                                            styles.podHeader
                                                        }
                                                    >
                                                        <h5
                                                            className={
                                                                styles.podName
                                                            }
                                                            title={pod.name}
                                                        >
                                                            {pod.name}
                                                            <span
                                                                className={
                                                                    styles.resourceTypeBadge
                                                                }
                                                            >
                                                                {formatResourceType(
                                                                    pod.resourceType,
                                                                )}
                                                            </span>
                                                        </h5>
                                                        <div
                                                            className={
                                                                styles.podControls
                                                            }
                                                        >
                                                            <button
                                                                className={
                                                                    styles.applyButton
                                                                }
                                                                onClick={() =>
                                                                    applyChanges(
                                                                        namespace,
                                                                        pod,
                                                                    )
                                                                }
                                                                disabled={
                                                                    isLoading
                                                                }
                                                            >
                                                                {isLoading ? (
                                                                    <>
                                                                        <svg
                                                                            width="16"
                                                                            height="16"
                                                                            fill="none"
                                                                            viewBox="0 0 24 24"
                                                                            stroke="currentColor"
                                                                            style={{
                                                                                animation:
                                                                                    "spin 1s linear infinite",
                                                                            }}
                                                                        >
                                                                            <path
                                                                                strokeLinecap="round"
                                                                                strokeLinejoin="round"
                                                                                strokeWidth={
                                                                                    2
                                                                                }
                                                                                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                                                                            />
                                                                        </svg>
                                                                        Applying...
                                                                    </>
                                                                ) : (
                                                                    <>
                                                                        <svg
                                                                            width="16"
                                                                            height="16"
                                                                            fill="none"
                                                                            viewBox="0 0 24 24"
                                                                            stroke="currentColor"
                                                                        >
                                                                            <path
                                                                                strokeLinecap="round"
                                                                                strokeLinejoin="round"
                                                                                strokeWidth={
                                                                                    2
                                                                                }
                                                                                d="M5 13l4 4L19 7"
                                                                            />
                                                                        </svg>
                                                                        Apply
                                                                        Changes
                                                                    </>
                                                                )}
                                                            </button>
                                                        </div>
                                                    </div>
                                                    <div
                                                        className={
                                                            styles.resourceInputs
                                                        }
                                                    >
                                                        <div
                                                            className={
                                                                styles.resourceInput
                                                            }
                                                        >
                                                            <label
                                                                className={
                                                                    styles.resourceLabel
                                                                }
                                                            >
                                                                CPU Limit
                                                                <button
                                                                    className={
                                                                        styles.infoButton
                                                                    }
                                                                    onMouseEnter={() =>
                                                                        showTooltip(
                                                                            `${pod.name}-cpuLimit`,
                                                                        )
                                                                    }
                                                                    onMouseLeave={
                                                                        hideTooltip
                                                                    }
                                                                >
                                                                    ?
                                                                </button>
                                                            </label>
                                                            <input
                                                                type="text"
                                                                className={
                                                                    styles.resourceInputField
                                                                }
                                                                value={
                                                                    values.cpuLimit ??
                                                                    ""
                                                                }
                                                                onChange={(e) =>
                                                                    handleChange(
                                                                        pod.name,
                                                                        "cpuLimit",
                                                                        e.target
                                                                            .value,
                                                                    )
                                                                }
                                                            />
                                                            {activeTooltip ===
                                                                `${pod.name}-cpuLimit` && (
                                                                <div
                                                                    className={
                                                                        styles.tooltip
                                                                    }
                                                                >
                                                                    Maximum CPU
                                                                    cores the
                                                                    container
                                                                    can use.
                                                                    Example: 0.5
                                                                    equals 500m
                                                                    (millicores).
                                                                </div>
                                                            )}
                                                        </div>
                                                        <div
                                                            className={
                                                                styles.resourceInput
                                                            }
                                                        >
                                                            <label
                                                                className={
                                                                    styles.resourceLabel
                                                                }
                                                            >
                                                                CPU Request
                                                                <button
                                                                    className={
                                                                        styles.infoButton
                                                                    }
                                                                    onMouseEnter={() =>
                                                                        showTooltip(
                                                                            `${pod.name}-cpuRequest`,
                                                                        )
                                                                    }
                                                                    onMouseLeave={
                                                                        hideTooltip
                                                                    }
                                                                >
                                                                    ?
                                                                </button>
                                                            </label>
                                                            <input
                                                                type="text"
                                                                className={
                                                                    styles.resourceInputField
                                                                }
                                                                value={
                                                                    values.cpuRequest ??
                                                                    ""
                                                                }
                                                                onChange={(e) =>
                                                                    handleChange(
                                                                        pod.name,
                                                                        "cpuRequest",
                                                                        e.target
                                                                            .value,
                                                                    )
                                                                }
                                                            />
                                                            {activeTooltip ===
                                                                `${pod.name}-cpuRequest` && (
                                                                <div
                                                                    className={
                                                                        styles.tooltip
                                                                    }
                                                                >
                                                                    Guaranteed
                                                                    CPU cores
                                                                    for the
                                                                    container.
                                                                    Example: 0.2
                                                                    equals 200m
                                                                    (millicores).
                                                                </div>
                                                            )}
                                                        </div>
                                                        <div
                                                            className={
                                                                styles.resourceInput
                                                            }
                                                        >
                                                            <label
                                                                className={
                                                                    styles.resourceLabel
                                                                }
                                                            >
                                                                Memory Limit
                                                                <button
                                                                    className={
                                                                        styles.infoButton
                                                                    }
                                                                    onMouseEnter={() =>
                                                                        showTooltip(
                                                                            `${pod.name}-memLimit`,
                                                                        )
                                                                    }
                                                                    onMouseLeave={
                                                                        hideTooltip
                                                                    }
                                                                >
                                                                    ?
                                                                </button>
                                                            </label>
                                                            <input
                                                                type="text"
                                                                className={
                                                                    styles.resourceInputField
                                                                }
                                                                value={
                                                                    values.memoryLimit ??
                                                                    ""
                                                                }
                                                                onChange={(e) =>
                                                                    handleChange(
                                                                        pod.name,
                                                                        "memoryLimit",
                                                                        e.target
                                                                            .value,
                                                                    )
                                                                }
                                                            />
                                                            {activeTooltip ===
                                                                `${pod.name}-memLimit` && (
                                                                <div
                                                                    className={
                                                                        styles.tooltip
                                                                    }
                                                                >
                                                                    Maximum
                                                                    memory the
                                                                    container
                                                                    can use.
                                                                    Example: 512
                                                                    (MB).
                                                                </div>
                                                            )}
                                                        </div>
                                                        <div
                                                            className={
                                                                styles.resourceInput
                                                            }
                                                        >
                                                            <label
                                                                className={
                                                                    styles.resourceLabel
                                                                }
                                                            >
                                                                Memory Request
                                                                <button
                                                                    className={
                                                                        styles.infoButton
                                                                    }
                                                                    onMouseEnter={() =>
                                                                        showTooltip(
                                                                            `${pod.name}-memRequest`,
                                                                        )
                                                                    }
                                                                    onMouseLeave={
                                                                        hideTooltip
                                                                    }
                                                                >
                                                                    ?
                                                                </button>
                                                            </label>
                                                            <input
                                                                type="text"
                                                                className={
                                                                    styles.resourceInputField
                                                                }
                                                                value={
                                                                    values.memoryRequest ??
                                                                    ""
                                                                }
                                                                onChange={(e) =>
                                                                    handleChange(
                                                                        pod.name,
                                                                        "memoryRequest",
                                                                        e.target
                                                                            .value,
                                                                    )
                                                                }
                                                            />
                                                            {activeTooltip ===
                                                                `${pod.name}-memRequest` && (
                                                                <div
                                                                    className={
                                                                        styles.tooltip
                                                                    }
                                                                >
                                                                    Guaranteed
                                                                    memory for
                                                                    the
                                                                    container.
                                                                    Example: 256
                                                                    (MB).
                                                                </div>
                                                            )}
                                                        </div>
                                                        <div
                                                            className={
                                                                styles.resourceInput
                                                            }
                                                        >
                                                            <label
                                                                className={
                                                                    styles.resourceLabel
                                                                }
                                                            >
                                                                Replicas
                                                                <button
                                                                    className={
                                                                        styles.infoButton
                                                                    }
                                                                    onMouseEnter={() =>
                                                                        showTooltip(
                                                                            `${pod.name}-replicas`,
                                                                        )
                                                                    }
                                                                    onMouseLeave={
                                                                        hideTooltip
                                                                    }
                                                                >
                                                                    ?
                                                                </button>
                                                            </label>
                                                            <input
                                                                type="text"
                                                                className={
                                                                    styles.resourceInputField
                                                                }
                                                                value={
                                                                    values.replicas ??
                                                                    ""
                                                                }
                                                                onChange={(e) =>
                                                                    handleChange(
                                                                        pod.name,
                                                                        "replicas",
                                                                        e.target
                                                                            .value,
                                                                    )
                                                                }
                                                            />
                                                            {activeTooltip ===
                                                                `${pod.name}-replicas` && (
                                                                <div
                                                                    className={
                                                                        styles.tooltip
                                                                    }
                                                                >
                                                                    Number of
                                                                    identical
                                                                    pods to run.
                                                                    Only applies
                                                                    to
                                                                    Deployments,
                                                                    StatefulSets,
                                                                    etc.
                                                                </div>
                                                            )}
                                                        </div>
                                                    </div>
                                                    <p
                                                        className={
                                                            styles.resourceTip
                                                        }
                                                    >
                                                        Status:{" "}
                                                        <span
                                                            className={`${styles.statusBadge} ${styles[`status${pod.status}`]}`}
                                                        >
                                                            {pod.status}
                                                        </span>
                                                        • Last Applied:{" "}
                                                        {new Date().toLocaleString()}
                                                    </p>
                                                </div>
                                            );
                                        })}
                                    </div>
                                </div>
                            ),
                        )}
                    </div>
                )
            )}

            <div className={styles.dashboardGrid}>
                <div className={styles.chartCard}>
                    <div className={styles.chartHeader}>
                        <h3 className={styles.chartTitle}>
                            <svg
                                width="20"
                                height="20"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
                                />
                            </svg>
                            CPU Usage
                        </h3>
                        <div className={styles.chartControls}>
                            <div className={styles.selectContainer}>
                                <select
                                    value={cpuIntervalMs}
                                    onChange={(e) =>
                                        setCpuIntervalMs(Number(e.target.value))
                                    }
                                >
                                    <option value={5 * 60 * 1000}>
                                        Last 5 min
                                    </option>
                                    <option value={30 * 60 * 1000}>
                                        Last 30 min
                                    </option>
                                    <option value={60 * 60 * 1000}>
                                        Last 1 hour
                                    </option>
                                    <option value={2 * 60 * 60 * 1000}>
                                        Last 2 hours
                                    </option>
                                    <option value={3 * 60 * 60 * 1000}>
                                        Last 3 hours
                                    </option>
                                    <option value={6 * 60 * 60 * 1000}>
                                        Last 6 hours
                                    </option>
                                </select>
                            </div>
                        </div>
                    </div>
                    <div className={styles.chartBody}>
                        <CpuChart
                            interval={cpuIntervalMs}
                            query="sum(rate(container_cpu_usage_seconds_total[1m])) by (namespace)"
                            label="CPU"
                            unitFormatter={(v) => `${(v * 100).toFixed(2)} %`}
                        />
                        <div className={styles.chartLegendDescription}>
                            Real-time CPU utilization by namespace (percentage)
                        </div>
                    </div>
                </div>

                <div className={styles.chartCard}>
                    <div className={styles.chartHeader}>
                        <h3 className={styles.chartTitle}>
                            <svg
                                width="20"
                                height="20"
                                fill="none"
                                viewBox="0 0 24 24"
                                stroke="currentColor"
                            >
                                <path
                                    strokeLinecap="round"
                                    strokeLinejoin="round"
                                    strokeWidth={2}
                                    d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
                                />
                            </svg>
                            Memory Usage
                        </h3>
                        <div className={styles.chartControls}>
                            <div className={styles.selectContainer}>
                                <select
                                    value={memIntervalMs}
                                    onChange={(e) =>
                                        setMemIntervalMs(Number(e.target.value))
                                    }
                                >
                                    <option value={5 * 60 * 1000}>
                                        Last 5 min
                                    </option>
                                    <option value={30 * 60 * 1000}>
                                        Last 30 min
                                    </option>
                                    <option value={60 * 60 * 1000}>
                                        Last 1 hour
                                    </option>
                                    <option value={2 * 60 * 60 * 1000}>
                                        Last 2 hours
                                    </option>
                                    <option value={3 * 60 * 60 * 1000}>
                                        Last 3 hours
                                    </option>
                                    <option value={6 * 60 * 60 * 1000}>
                                        Last 6 hours
                                    </option>
                                </select>
                            </div>
                        </div>
                    </div>
                    <div className={styles.chartBody}>
                        <CpuChart
                            interval={memIntervalMs}
                            query="sum(container_memory_usage_bytes) by (namespace)"
                            label="Memory"
                            unitFormatter={(v) =>
                                `${(v / 1024 / 1024).toFixed(1)} MB`
                            }
                        />
                        <div className={styles.chartLegendDescription}>
                            Real-time memory consumption by namespace
                            (megabytes)
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};
