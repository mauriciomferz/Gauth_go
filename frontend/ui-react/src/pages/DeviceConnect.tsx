import React, { useState } from 'react';
import {
    makeStyles,
    Input,
    Button,
    Title1,
    Body1,
    Card,
    CardHeader,
    Spinner,
    MessageBar,
    MessageBarTitle,
    MessageBarBody,
    MessageBarIntent,
    shorthands,
} from '@fluentui/react-components';
import { DeviceMeetingRoom24Regular } from '@fluentui/react-icons';
import { useSearchParams } from 'react-router-dom';

const useStyles = makeStyles({
    container: {
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '80vh',
        ...shorthands.padding('20px'),
    },
    card: {
        width: '100%',
        maxWidth: '400px',
        textAlign: 'center',
        ...shorthands.padding('24px'),
    },
    input: {
        width: '100%',
        marginBottom: '20px',
        '& input': {
            textAlign: 'center',
            fontSize: '24px',
            letterSpacing: '4px',
            textTransform: 'uppercase',
        },
    },
    button: {
        width: '100%',
        marginTop: '16px',
    },
    icon: {
        fontSize: '64px',
        color: '#0078d4',
        marginBottom: '24px',
    },
    message: {
        marginTop: '16px',
        textAlign: 'left',
    },
});

export const DeviceConnect: React.FC = () => {
    const styles = useStyles();
    const [searchParams] = useSearchParams();
    const [userCode, setUserCode] = useState(searchParams.get('code') || '');
    const [loading, setLoading] = useState(false);
    const [result, setResult] = useState<{ type: MessageBarIntent; message: string } | null>(null);
    const [isSuccess, setIsSuccess] = useState(false);

    const formatCode = (val: string) => {
        // Basic formatting XXXX-XXXX
        const raw = val.replace(/[^A-Za-z0-9]/g, '').toUpperCase();
        if (raw.length > 4) {
            return raw.slice(0, 4) + '-' + raw.slice(4, 9);
        }
        return raw;
    };

    const handleInputChange = (_ev: React.ChangeEvent<HTMLInputElement>, data: { value: string }) => {
        setUserCode(formatCode(data.value));
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (userCode.length < 9) return;

        setLoading(true);
        setResult(null);

        try {
            // Assuming Vite proxy directs /device/verify to backend
            const params = new URLSearchParams();
            params.append('user_code', userCode);
            params.append('action', 'authorize');
            params.append('user_id', 'test-user-123'); // Simulate authenticated user

            const response = await fetch('/device/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: params,
            });

            const data = await response.json();

            if (response.ok) {
                setIsSuccess(true);
                setResult({
                    type: 'success',
                    message: 'Device connected successfully! You can close this window now.',
                });
            } else {
                setResult({
                    type: 'error',
                    message: data.error_description || 'Verification failed. Please check the code and try again.',
                });
            }
        } catch (error) {
            setResult({
                type: 'error',
                message: 'Network error. Please try again.',
            });
        } finally {
            setLoading(false);
        }
    };

    if (isSuccess) {
        return (
            <div className={styles.container}>
                <Card className={styles.card}>
                    <div style={{ display: 'flex', justifyContent: 'center', marginBottom: '20px' }}>
                        <DeviceMeetingRoom24Regular style={{ fontSize: '64px', color: 'green' }} />
                    </div>
                    <Title1>Success!</Title1>
                    <Body1 style={{ marginTop: '20px', marginBottom: '40px' }}>
                        You have successfully authorized the device.
                    </Body1>
                    <Button appearance="primary" onClick={() => { setIsSuccess(false); setUserCode(''); setResult(null); }}>
                        Connect Another Device
                    </Button>
                </Card>
            </div>
        );
    }

    return (
        <div className={styles.container}>
            <Card className={styles.card}>
                <div style={{ display: 'flex', justifyContent: 'center' }}>
                    <DeviceMeetingRoom24Regular className={styles.icon} />
                </div>
                <CardHeader
                    header={<Title1>Connect a Device</Title1>}
                    description={<Body1>Enter the code displayed on your device</Body1>}
                />

                <form onSubmit={handleSubmit} style={{ width: '100%', marginTop: '24px' }}>
                    <Input
                        className={styles.input}
                        value={userCode}
                        onChange={handleInputChange}
                        placeholder="XXXX-XXXX"
                        maxLength={9}
                        size="large"
                        disabled={loading}
                    />

                    <Button
                        type="submit"
                        appearance="primary"
                        className={styles.button}
                        size="large"
                        disabled={loading || userCode.length < 8}
                    >
                        {loading ? <Spinner size="tiny" /> : 'Next'}
                    </Button>
                </form>

                {result && (
                    <MessageBar intent={result.type} className={styles.message}>
                        <MessageBarBody>
                            <MessageBarTitle>{result.type === 'error' ? 'Error' : 'Success'}</MessageBarTitle>
                            {result.message}
                        </MessageBarBody>
                    </MessageBar>
                )}
            </Card>
        </div>
    );
};
