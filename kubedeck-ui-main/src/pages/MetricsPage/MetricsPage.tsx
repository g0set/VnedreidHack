import React, { useEffect, useState } from "react";
import CpuChart from "../../coms/grafics/CpuChart";
import styles from "./MetricsPage.module.css";

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

interface dataToFetch {
  namespace: string,
  resource: {
    resourceType: string,
    name: string,
    cpuRequest: number,
    memoryRequest: number,
    cpuLimit: number,
    memoryLimit: number,
    replicas: number
  }
}


export const MetricsPage: React.FC = () => {
  const [data, setData] = useState<ApiData | null>(null);
  const [podValues, setPodValues] = useState<Record<string, PodValues>>({});
  const [cpuIntervalMs, setCpuIntervalMs] = useState(5 * 60 * 1000);
  const [memIntervalMs, setMemIntervalMs] = useState(5 * 60 * 1000);
  const [isLoading, setIsLoading] = useState<boolean>(false)


  useEffect(() => {
    fetch("https://ui.orion.nikcorp.ru/operator/analyze/resources")
      .then((res) => res.json())
      .then((json: ApiData) => {
        setData(json);
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
      })
      .catch((err) => {
        console.error("Ошибка при загрузке метрик:", err);
      });
  }, []);

  const handleChange = (podName: string, field: keyof PodValues, value: string) => {
    setPodValues((prev) => ({
      ...prev,
      [podName]: {
        ...prev[podName],
        [field]: Number(value),
      },
    }));
  };

  const fetchResize = async (data: dataToFetch) => {
    try {
      setIsLoading(true);

      const response = await fetch("https://ui.orion.nikcorp.ru/operator/updatefromfront", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ namespace: data.namespace, resource: data.resource }),
      });

      if (!response.ok) throw new Error("Failed to fetch stats");

    } finally {

      setIsLoading(false);
    }
  };

  return (
    <div className={styles.zoomContainer}>
      <div className={styles.metrics}>
        <div className={styles.metrics_metric}>
          <h1 className={styles.metric__title}>Рекомендации</h1>
          <div className={styles.metric_recs}>
            {data ? (
              <>
                <p className={styles.recs__descr}>{data.message}</p>
                <div className={styles.recs_namespaces}>
                  {Object.entries(data.namespaces).map(([namespace, pods]) => (
                    <div key={namespace} className={styles.metrics_namespace}>
                      <p className={styles.namespace__descr}>{namespace}</p>
                      <div className={styles.recs_pods}>
                        {pods.map((pod) => {
                          const values = podValues[pod.name] || {};
                          return (
                            <div key={pod.name} className={styles.pods_pod}>
                              <p className={styles.pod__descr} title={pod.name}>{pod.name}</p>
                              <div className={styles.pod_changin}>
                                <div className={styles.changin_inputs}>
                                  <input
                                    className={styles.inputs__input}
                                    onChange={(e) => handleChange(pod.name, "cpuLimit", e.target.value)}
                                    type="text"
                                    value={values.cpuLimit ?? ""}
                                  />
                                  <input
                                    className={styles.inputs__input}
                                    onChange={(e) => handleChange(pod.name, "cpuRequest", e.target.value)}
                                    type="text"
                                    value={values.cpuRequest ?? ""}
                                  />
                                  <input
                                    className={styles.inputs__input}
                                    onChange={(e) => handleChange(pod.name, "memoryLimit", e.target.value)}
                                    type="text"
                                    value={values.memoryLimit ?? ""}
                                  />
                                  <input
                                    className={styles.inputs__input}
                                    onChange={(e) => handleChange(pod.name, "memoryRequest", e.target.value)}
                                    type="text"
                                    value={values.memoryRequest ?? ""}
                                  />
                                  <input
                                    className={styles.inputs__input}
                                    onChange={(e) => handleChange(pod.name, "replicas", e.target.value)}
                                    type="text"
                                    value={values.replicas ?? ""}
                                  />
                                </div>
                                <button className={styles.changin__button} onClick={() =>
                                  fetchResize({
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
                                  })
                                }>{isLoading ? '(*)' : 'Применить'}</button>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <div className={styles.loadingContainer}>
                <div className={styles.spinner} />
                <h1 className={styles.loading}>Отчет по рекомендациям генерируется...</h1>
              </div>
            )}
          </div>
        </div>

        <div className={styles.metrics_graph}>
          <h1 className={styles.metric__title}>CPU Usage</h1>
          <div className={styles.controls}>
            <label htmlFor="cpu-interval">Интервал (CPU):</label>
            <select
              id="cpu-interval"
              value={cpuIntervalMs}
              onChange={(e) => setCpuIntervalMs(Number(e.target.value))}
            >
              <option value={5 * 60 * 1000}>5 минут</option>
              <option value={30 * 60 * 1000}>30 минут</option>
              <option value={60 * 60 * 1000}>1 час</option>
              <option value={2 * 60 * 60 * 1000}>2 часа</option>
              <option value={3 * 60 * 60 * 1000}>3 часа</option>
              <option value={6 * 60 * 60 * 1000}>6 часов</option>
            </select>
          </div>

          <div className={styles.chartWrapper}>
            <CpuChart
              interval={cpuIntervalMs}
              query="sum(rate(container_cpu_usage_seconds_total[1m])) by (namespace)"
              label="CPU"
              unitFormatter={(v) => `${(v * 100).toFixed(2)} %`}
            />
          </div>
        </div>

        <div className={styles.metrics_graph}>
          <h1 className={styles.metric__title}>Memory Usage</h1>
          <div className={styles.controls}>
            <label htmlFor="mem-interval">Интервал (Memory):</label>
            <select
              id="mem-interval"
              value={memIntervalMs}
              onChange={(e) => setMemIntervalMs(Number(e.target.value))}
            >
              <option value={5 * 60 * 1000}>5 минут</option>
              <option value={30 * 60 * 1000}>30 минут</option>
              <option value={60 * 60 * 1000}>1 час</option>
              <option value={2 * 60 * 60 * 1000}>2 часа</option>
              <option value={3 * 60 * 60 * 1000}>3 часа</option>
              <option value={6 * 60 * 60 * 1000}>6 часов</option>
            </select>
          </div>

          <div className={styles.chartWrapper}>
            <CpuChart
              interval={memIntervalMs}
              query="sum(container_memory_usage_bytes) by (namespace)"
              label="Memory"
              unitFormatter={(v) => `${(v / 1024 / 1024).toFixed(1)} MB`}
            />
          </div>
        </div>
      </div>
    </div>
  );
};
