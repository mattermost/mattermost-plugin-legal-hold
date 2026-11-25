import {useState} from 'react';

import ConfirmDialog from '@/components/confirm_dialog';
import type {LegalHold} from '@/types';

interface ConfirmReleaseProps {
    legalHold: LegalHold | null;
    releaseLegalHold: (id: string) => Promise<unknown>;
    onClose: () => void;
    visible: boolean;
}

const ConfirmRelease = (props: ConfirmReleaseProps) => {
    const [saving, setSaving] = useState(false);
    const [serverError, setServerError] = useState('');

    const release = () => {
        if (!props.legalHold) {
            return;
        }

        setSaving(true);
        props.releaseLegalHold(props.legalHold.id).then(() => {
            setSaving(false);
            setServerError('');
            props.onClose();
        }).catch((error) => {
            setSaving(false);
            setServerError(String(error));
        });
    };

    const onCancel = () => {
        setServerError('');
        props.onClose();
    };

    return (
        <ConfirmDialog
            open={props.visible}
            title='Release legal hold'
            message={props.legalHold && (
                <>
                    {'Are you sure you want to release the legal hold '}
                    <strong>{'"'}{props.legalHold.display_name}{'"'}</strong>
                    {'? All data associated with it will immediately be deleted and cannot be recovered.'}
                </>
            )}
            confirmLabel='Release'
            destructive={true}
            loading={saving}
            error={serverError}
            onConfirm={release}
            onCancel={onCancel}
        />
    );
};

export default ConfirmRelease;
