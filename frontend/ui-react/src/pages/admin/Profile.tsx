import { toast } from 'sonner';
import {
    makeStyles,
    tokens,
    Card,
    Text,
    Title2,
    Persona,
    Button,
} from '@fluentui/react-components';

const useStyles = makeStyles({
    container: {
        display: 'flex',
        flexDirection: 'column',
        gap: '24px',
        maxWidth: '800px',
        margin: '0 auto',
    },
    card: {
        padding: '24px',
        display: 'flex',
        flexDirection: 'column',
        gap: '20px',
    },
    sectionTitle: {
        marginBottom: '12px',
        color: tokens.colorNeutralForeground1,
    },
    fieldGroup: {
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
    },
    label: {
        color: tokens.colorNeutralForeground2,
        fontSize: '14px',
    },
    value: {
        color: tokens.colorNeutralForeground1,
        fontSize: '16px',
        fontWeight: 500,
    },
    actions: {
        paddingTop: '20px',
        borderTop: `1px solid ${tokens.colorNeutralStroke2}`,
        display: 'flex',
        gap: '12px',
    },
});

export default function Profile() {
    const classes = useStyles();
    const userName = localStorage.getItem('admin_username') || 'Admin User';
    const userRole = localStorage.getItem('admin_role') || 'admin';
    const tenantId = 'test-tenant-1'; // Placeholder

    return (
        <div className={classes.container}>
            <Title2>User Profile</Title2>

            <Card className={classes.card}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '16px', marginBottom: '16px' }}>
                    <Persona
                        name={userName}
                        secondaryText={userRole}
                        size="huge"
                    />
                </div>

                <div className={classes.fieldGroup}>
                    <Text className={classes.label}>Full Name</Text>
                    <Text className={classes.value}>{userName}</Text>
                </div>

                <div className={classes.fieldGroup}>
                    <Text className={classes.label}>Role</Text>
                    <Text className={classes.value}>{userRole.charAt(0).toUpperCase() + userRole.slice(1)}</Text>
                </div>

                <div className={classes.fieldGroup}>
                    <Text className={classes.label}>Tenant ID</Text>
                    <Text className={classes.value}>{tenantId}</Text>
                </div>

                <div className={classes.actions}>
                    <Button appearance="primary" onClick={() => toast.info('Edit Profile functionality is simulated in Development Mode.', {
                        description: 'In a production environment, this would open a profile editing form.',
                        duration: 5000,
                    })}>
                        Edit Profile
                    </Button>
                    <Button appearance="secondary" onClick={() => toast.error('Password changes are disabled in Development Mode.', {
                        description: 'Please contact a system administrator for password resets in this environment.',
                        duration: 5000,
                    })}>
                        Change Password
                    </Button>
                </div>
            </Card>

            <Card className={classes.card}>
                <Title2 className={classes.sectionTitle}>Preferences</Title2>
                <Text>Notification settings and other user preferences will appear here.</Text>
            </Card>
        </div>
    );
}
