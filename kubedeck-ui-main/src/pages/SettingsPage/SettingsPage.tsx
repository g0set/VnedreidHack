import React, { useState } from "react";
import styles from "./SettingsPage.module.css";

interface TelegramBotConfig {
    token?: string;
    checkInterval?: number;
    chatIDs?: number[];
    responseStyle?: string;
}

interface CloudConfig {
    cloudProvider: 'timeweb' | 'yandex';
    timewebToken?: string;
    timewebClusterId?: string;
    yandexToken?: string;
    yandexFolderId?: string;
}

interface LlmConfig {
    hostname?: string;
    token?: string;
}

interface ServerResponse {
    success: boolean;
    message: string;
    changed: boolean;
    settings?: {
        tokenUpdated: boolean;
        intervalUpdated: boolean;
        chatIDsUpdated: boolean;
        currentInterval: number;
        currentChatIDs: number[];
    };
}

export const SettingsPage: React.FC = () => {
    const [telegramConfig, setTelegramConfig] = useState<TelegramBotConfig>({});
    const [cloudConfig, setCloudConfig] = useState<CloudConfig>({
        cloudProvider: 'timeweb',
        timewebToken: '',
        timewebClusterId: '',
        yandexToken: '',
        yandexFolderId: ''
    });
    const [llmConfig, setLlmConfig] = useState<LlmConfig>({
        hostname: '',
        token: ''
    });
    const [message, setMessage] = useState<{
        text: string;
        type: "success" | "error";
    } | null>(null);
    const [newChatId, setNewChatId] = useState<string>("");

    const handleTelegramTokenChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setTelegramConfig({ ...telegramConfig, token: e.target.value });
    };

    const handleIntervalChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        const value = parseInt(e.target.value);
        if (!isNaN(value) && value > 0) {
            setTelegramConfig({ ...telegramConfig, checkInterval: value });
        }
    };

    const handleResponseStyleChange = (
        e: React.ChangeEvent<HTMLTextAreaElement>,
    ) => {
        setTelegramConfig({ ...telegramConfig, responseStyle: e.target.value });
    };

    const handleAddChatId = (e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === "Enter" && newChatId.trim()) {
            const chatId = parseInt(newChatId.trim());
            if (!isNaN(chatId)) {
                setTelegramConfig({
                    ...telegramConfig,
                    chatIDs: [...(telegramConfig.chatIDs || []), chatId],
                });
                setNewChatId("");
            }
        }
    };

    const handleRemoveChatId = (index: number) => {
        const newChatIDs = [...(telegramConfig.chatIDs || [])];
        newChatIDs.splice(index, 1);
        setTelegramConfig({ ...telegramConfig, chatIDs: newChatIDs });
    };
    
    const handleCloudProviderChange = (provider: 'timeweb' | 'yandex') => {
        setCloudConfig({ ...cloudConfig, cloudProvider: provider });
    };
    
    const handleTimewebTokenChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setCloudConfig({ ...cloudConfig, timewebToken: e.target.value });
    };
    
    const handleTimewebClusterIdChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setCloudConfig({ ...cloudConfig, timewebClusterId: e.target.value });
    };
    
    const handleYandexTokenChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setCloudConfig({ ...cloudConfig, yandexToken: e.target.value });
    };
    
    const handleYandexFolderIdChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setCloudConfig({ ...cloudConfig, yandexFolderId: e.target.value });
    };
    
    const handleLlmHostnameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setLlmConfig({ ...llmConfig, hostname: e.target.value });
    };
    
    const handleLlmTokenChange = (e: React.ChangeEvent<HTMLInputElement>) => {
        setLlmConfig({ ...llmConfig, token: e.target.value });
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setMessage(null);

        try {
            // This is just a mock - in a real implementation we would send all configs
            const response = await fetch(
                "https://ui.orion.nikcorp.ru/operator/telegram/config",
                {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify(telegramConfig),
                },
            );

            const data: ServerResponse = await response.json();

            if (data.success) {
                setMessage({ text: "All configurations successfully updated", type: "success" });
            } else {
                setMessage({ text: data.message, type: "error" });
            }
        } catch {
            setMessage({
                text: "Failed to update configuration. Please try again.",
                type: "error",
            });
        }
    };

    return (
        <div className={styles.settings}>
            <h1 className={styles.title}>Настройки системы</h1>
            <form onSubmit={handleSubmit} className={styles.form}>
                <h2 className={styles.subtitle}>Настройки Telegram бота</h2>
                <div className={styles.formGroup}>
                    <label className={styles.label}>Token</label>
                    <input
                        type="password"
                        value={telegramConfig.token || ""}
                        onChange={handleTelegramTokenChange}
                        className={styles.input}
                        placeholder="Введите token бота"
                    />
                </div>

                <div className={styles.formGroup}>
                    <label className={styles.label}>
                        Интервал проверки (в секундах)
                    </label>
                    <input
                        type="number"
                        value={telegramConfig.checkInterval || ""}
                        onChange={handleIntervalChange}
                        className={styles.input}
                        min="1"
                        placeholder="Например: 3600"
                    />
                </div>

                <div className={styles.formGroup}>
                    <label className={styles.label}>ID чатов</label>
                    <div className={styles.chatIdsContainer}>
                        <div className={styles.chipInput}>
                            {telegramConfig.chatIDs?.map((chatId, index) => (
                                <div key={index} className={styles.chip}>
                                    {chatId}
                                    <span
                                        className={styles.deleteChip}
                                        onClick={() =>
                                            handleRemoveChatId(index)
                                        }
                                    >
                                        ✕
                                    </span>
                                </div>
                            ))}
                            <input
                                type="text"
                                value={newChatId}
                                onChange={(e) => setNewChatId(e.target.value)}
                                onKeyDown={handleAddChatId}
                                className={styles.input}
                                placeholder="Введите ID чата и нажмите Enter"
                            />
                        </div>
                    </div>
                </div>

                <div className={styles.formGroup}>
                    <label className={styles.label}>Стиль ответов</label>
                    <textarea
                        value={telegramConfig.responseStyle || ""}
                        onChange={handleResponseStyleChange}
                        className={`${styles.input} ${styles.textArea}`}
                        placeholder="Введите стиль ответов бота"
                    />
                </div>

                <h2 className={styles.subtitle}>Настройки облачного провайдера</h2>
                
                <div className={styles.formGroup}>
                    <label className={styles.label}>Выберите провайдера</label>
                    <div className={styles.radioGroup}>
                        <label className={styles.radioLabel}>
                            <input
                                type="radio"
                                checked={cloudConfig.cloudProvider === 'timeweb'}
                                onChange={() => handleCloudProviderChange('timeweb')}
                                className={styles.radioInput}
                            />
                            Timeweb
                        </label>
                        <label className={styles.radioLabel}>
                            <input
                                type="radio"
                                checked={cloudConfig.cloudProvider === 'yandex'}
                                onChange={() => handleCloudProviderChange('yandex')}
                                className={styles.radioInput}
                            />
                            Yandex Cloud
                        </label>
                    </div>
                </div>

                {cloudConfig.cloudProvider === 'timeweb' ? (
                    <>
                        <div className={styles.formGroup}>
                            <label className={styles.label}>Timeweb API Token</label>
                            <input
                                type="password"
                                value={cloudConfig.timewebToken || ""}
                                onChange={handleTimewebTokenChange}
                                className={styles.input}
                                placeholder="Введите токен API Timeweb"
                            />
                        </div>

                        <div className={styles.formGroup}>
                            <label className={styles.label}>ID кластера Timeweb</label>
                            <input
                                type="text"
                                value={cloudConfig.timewebClusterId || ""}
                                onChange={handleTimewebClusterIdChange}
                                className={styles.input}
                                placeholder="Введите ID кластера"
                            />
                        </div>
                    </>
                ) : (
                    <>
                        <div className={styles.formGroup}>
                            <label className={styles.label}>Токен Yandex Cloud</label>
                            <input
                                type="password"
                                value={cloudConfig.yandexToken || ""}
                                onChange={handleYandexTokenChange}
                                className={styles.input}
                                placeholder="Введите токен Yandex Cloud"
                            />
                        </div>

                        <div className={styles.formGroup}>
                            <label className={styles.label}>Folder ID Yandex Cloud</label>
                            <input
                                type="text"
                                value={cloudConfig.yandexFolderId || ""}
                                onChange={handleYandexFolderIdChange}
                                className={styles.input}
                                placeholder="Введите ID папки Yandex Cloud"
                            />
                        </div>
                    </>
                )}
                
                <h2 className={styles.subtitle}>Настройки LLM</h2>
                
                <div className={styles.formGroup}>
                    <label className={styles.label}>Hostname</label>
                    <input
                        type="text"
                        value={llmConfig.hostname || ""}
                        onChange={handleLlmHostnameChange}
                        className={styles.input}
                        placeholder="Введите hostname LLM сервера"
                    />
                </div>
                
                <div className={styles.formGroup}>
                    <label className={styles.label}>Token</label>
                    <input
                        type="password"
                        value={llmConfig.token || ""}
                        onChange={handleLlmTokenChange}
                        className={styles.input}
                        placeholder="Введите token LLM API"
                    />
                </div>

                <button type="submit" className={styles.button}>
                    Сохранить все настройки
                </button>

                {message && (
                    <div
                        className={`${styles.message} ${styles[message.type]}`}
                    >
                        {message.text}
                    </div>
                )}
            </form>
        </div>
    );
};
