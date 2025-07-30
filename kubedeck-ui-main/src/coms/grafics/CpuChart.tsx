// CpuChart.tsx
import React, { useEffect, useRef, useState } from "react";
import {
    LineChart,
    Line,
    XAxis,
    YAxis,
    Tooltip,
    CartesianGrid,
    ResponsiveContainer,
} from "recharts";
import styles from "./CpuChart.module.css";

interface CpuDataPoint {
    timestamp: number;
    [namespace: string]: number;
}

interface CpuChartProps {
    interval: number;
    query: string;
    label: string;
    unitFormatter?: (val: number) => string;
}

const PROMETHEUS_URL = "https://ui.orion.nikcorp.ru/prometheus";
const colors = ["#97BF69", "#F2CC0C", "#FF8C00", "#b877d9", "#FF3366", "#4682B4", "#00BFFF"];

const fetchDataRange = async (
    query: string,
    start: number,
    end: number,
    step: number
): Promise<CpuDataPoint[]> => {
    const url = `${PROMETHEUS_URL}/api/v1/query_range?query=${encodeURIComponent(
        query
    )}&start=${Math.floor(start / 1000)}&end=${Math.floor(end / 1000)}&step=${Math.floor(step / 1000)}`;

    try {
        const res = await fetch(url);
        const json = await res.json();

        const results: CpuDataPoint[] = [];
        const timestamps = json.data.result[0]?.values?.map((v: any) => Number(v[0]) * 1000) || [];

        timestamps.forEach((ts: number, i: number) => {
            const point: CpuDataPoint = { timestamp: ts };
            json.data.result.forEach((series: any) => {
                const ns = series.metric.namespace;
                const val = parseFloat(series.values[i]?.[1]);
                if (ns && !isNaN(val)) {
                    point[ns] = +val.toFixed(5);
                }
            });
            results.push(point);
        });

        return results;
    } catch (err) {
        console.error("Prometheus range fetch failed:", err);
        return [];
    }
};

const fetchLatestData = async (query: string): Promise<CpuDataPoint> => {
    const timestamp = Date.now();
    const url = `${PROMETHEUS_URL}/api/v1/query?query=${encodeURIComponent(query)}`;

    try {
        const res = await fetch(url);
        const json = await res.json();

        const point: CpuDataPoint = { timestamp };

        if (json.status === "success" && Array.isArray(json.data.result)) {
            json.data.result.forEach((entry: any) => {
                const ns = entry.metric.namespace;
                const val = parseFloat(entry.value?.[1]);
                if (ns && !isNaN(val)) {
                    point[ns] = +val.toFixed(5);
                }
            });
        }

        return point;
    } catch (err) {
        console.error("Prometheus fetch failed:", err);
        return { timestamp };
    }
};

const getMinuteTicks = (data: CpuDataPoint[]): number[] => {
    const seen = new Set<string>();
    const ticks: number[] = [];

    for (const point of data) {
        const date = new Date(point.timestamp);
        const key = `${date.getHours()}:${date.getMinutes()}`;
        if (!seen.has(key)) {
            seen.add(key);
            ticks.push(point.timestamp);
        }
    }

    return ticks;
};

const CpuChart: React.FC<CpuChartProps> = ({ interval, query, label, unitFormatter }) => {
    const [data, setData] = useState<CpuDataPoint[]>([]);
    const [sources, setSources] = useState<string[]>([]);
    const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

    useEffect(() => {
        const loadRangeData = async () => {
            const end = Date.now();
            const start = end - interval;
            const step = Math.max(10_000, Math.floor(interval / 60)); // макс 60 точек

            const points = await fetchDataRange(query, start, end, step);
            setData(points);

            const last = points[points.length - 1];
            if (last) {
                const namespaces = Object.keys(last).filter(k => k !== "timestamp");
                setSources(namespaces);
            }
        };

        loadRangeData();

        timerRef.current = setInterval(async () => {
            const newPoint = await fetchLatestData(query);
            const cutoff = Date.now() - interval;

            setData(prev => {
                const updated = [...prev, newPoint];
                return updated.filter(p => p.timestamp >= cutoff);
            });

            const keys = Object.keys(newPoint).filter(k => k !== "timestamp");
            if (keys.length > 0) setSources(keys);
        }, 10_000);

        return () => {
            if (timerRef.current) clearInterval(timerRef.current);
        };
    }, [interval, query]);

    return (
        <div className={styles.cpuChartContainer}>
            <div className={styles.chartArea}>
                <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={data}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                        <XAxis
                            dataKey="timestamp"
                            type="number"
                            scale="time"
                            stroke="#888"
                            domain={["dataMin", "dataMax"]}
                            ticks={getMinuteTicks(data)}
                            tickFormatter={(value) =>
                                new Date(value).toLocaleTimeString([], {
                                    hour: "2-digit",
                                    minute: "2-digit",
                                })
                            }
                        />
                        <YAxis domain={[0, "auto"]} stroke="#888" tickFormatter={unitFormatter} />
                        <Tooltip
                            labelFormatter={(value) =>
                                new Date(value as number).toLocaleTimeString([], {
                                    hour: "2-digit",
                                    minute: "2-digit",
                                    second: "2-digit",
                                })
                            }
                            formatter={(value) => [
                                unitFormatter ? unitFormatter(value as number) : value,
                                label,
                            ]}
                        />
                        {sources.map((source, index) => (
                            <Line
                                key={source}
                                type="monotone"
                                dataKey={source}
                                stroke={colors[index % colors.length]}
                                dot={false}
                                strokeWidth={2}
                                isAnimationActive={false}
                            />
                        ))}
                    </LineChart>
                </ResponsiveContainer>
            </div>

            <div className={styles.legend}>
                {sources.map((source, index) => (
                    <div
                        key={source}
                        className={styles.legendItem}
                        style={{ color: colors[index % colors.length] }}
                    >
                        ● {source}
                    </div>
                ))}
            </div>
        </div>
    );
};

export default CpuChart;
