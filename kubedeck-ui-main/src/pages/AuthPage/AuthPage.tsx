import React, { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import styles from "./AuthPage.module.css";

export const AuthPage: React.FC = () => {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [remember, setRemember] = useState(false);
    const [error, setError] = useState("");
    const navigate = useNavigate();

    useEffect(() => {
        const savedUser = localStorage.getItem("rememberUser");
        if (savedUser) {
            setUsername(savedUser);
        }
    }, []);

    const handleLogin = () => {
        if (username === "admin" && password === "slavyane_goida") {
            if (remember) {
                localStorage.setItem("rememberUser", username);
            } else {
                localStorage.removeItem("rememberUser");
            }
            navigate("/metrics");
        } else {
            setError("Неверный логин или пароль");
            setTimeout(() => setError(""), 3000);
        }
    };

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === "Enter") {
            handleLogin();
        }
    };

    return (
        <div className={styles.container}>
            <div className={styles.box}>
                <svg
                    className={styles.logo}
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 328 80"
                >
                    <g>
                        <path d="M0,24.9C0,18.5,2.5,12.5,7,8s10.5-7,16.9-7H79v12.2H23.9c-3.1,0-6.1,1.2-8.3,3.5c-2.2,2.2-3.4,5.2-3.4,8.3   l-0.1,20.4c-0.6,5.4-4.6,9.8-9.9,10.6L0,56.2V24.9z" />
                        <path d="M0,79.9V67.8h55.2c3.1,0,6.1-1.2,8.3-3.5c2.2-2.2,3.5-5.2,3.5-8.3V36.7c0-5.8,4.3-10.8,10-11.6l2.1-0.3V56   c0,6.4-2.5,12.4-7,16.9s-10.5,7-16.9,7H0z" />
                        <path d="M46.7,56.2V40.6l-7.2,8.6l-7.2-8.6v15.7h-9.5v-31h9l7.7,9.9l7.7-9.9h9v30.9C56.2,56.2,46.7,56.2,46.7,56.2z" />
                        <path d="M135.5,67.4h-13.6v-41H106v-12h45.5v12h-16V67.4z M201.9,67.4H161v-53h40.3v12h-26.7v7.2h21.2v11.5h-21.2v10.4   h27.3L201.9,67.4L201.9,67.4z M238.6,68.5c-8.5,0-15.5-2.6-21.1-7.7c-5.6-5.2-8.4-11.8-8.4-19.8s2.8-14.7,8.3-19.8   c5.6-5.2,12.6-7.7,21.1-7.7c5,0,9.6,1,13.7,3v13.4c-4.1-2.6-8.7-3.9-13.6-3.9c-4.4,0-8.2,1.4-11.1,4.2c-3,2.8-4.5,6.4-4.5,10.8   s1.5,8.1,4.5,10.8c3.1,2.8,6.9,4.2,11.6,4.2c5.6,0,10.2-1.4,13.9-4.2v13.5C248.8,67.4,244,68.5,238.6,68.5z M277.6,67.4H264v-53   h13.6v18.9h20V14.4h13.6v53h-13.6v-22h-20V67.4z" />
                    </g>
                </svg>

                <h2 className={styles.title}>Вход в систему</h2>

                {error && <div className={styles.errorMessage}>{error}</div>}

                <div className={styles.inputGroup}>
                    <input
                        type="text"
                        placeholder="Логин"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        className={styles.input}
                        onKeyDown={handleKeyDown}
                    />
                </div>

                <div className={styles.inputGroup}>
                    <input
                        type="password"
                        placeholder="Пароль"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        className={styles.input}
                        onKeyDown={handleKeyDown}
                    />
                </div>

                <div className={styles.checkboxContainer}>
                    <input
                        type="checkbox"
                        id="remember"
                        checked={remember}
                        onChange={(e) => setRemember(e.target.checked)}
                    />
                    <label htmlFor="remember" className={styles.checkboxLabel}>
                        Запомнить меня
                    </label>
                </div>

                <button onClick={handleLogin} className={styles.button}>
                    Войти
                </button>
            </div>
        </div>
    );
};
