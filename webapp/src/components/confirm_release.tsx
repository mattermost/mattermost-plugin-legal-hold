import React, {useState} from 'react';

import {UserProfile} from 'mattermost-redux/types/users';

import UsersInput from '@/components/users_input';
import {CreateLegalHold, LegalHold} from '@/types';
import {GenericModal} from '@/components/mattermost-webapp/generic_modal/generic_modal';
import Input from '@/components/mattermost-webapp/input/input';

import './create_legal_hold_form.scss';

interface ConfirmReleaseProps {
    legalHold: LegalHold;
    releaseLegalHold: (id: string) => Promise<any>;
    onExited: () => void;
    visible: boolean;
}

const ConfirmRelease = (props: ConfirmReleaseProps) => {
    const [saving, setSaving] = useState(false);
    const [serverError, setServerError] = useState('');

    const release = () => {
        setSaving(true);
        props.releaseLegalHold(props.legalHold.id).then((response) => {
            props.onExited();
        }).catch((error) => {
            setSaving(false);
            setServerError(error.toString());
        });
    };

    const onCancel = () => {
        props.onExited();
    };

    return (
        <GenericModal
            id='confirm-release-legal-hold-modal'
            className='confirm-release-legal-hold-modal'
            modalHeaderText='Release legal hold'
            confirmButtonText='Release'
            cancelButtonText='Cancel'
            errorText={serverError}
            autoCloseOnConfirmButton={false}
            compassDesign={true}
            isConfirmDisabled={saving}
            handleConfirm={release}
            handleCancel={onCancel}
            onExited={onCancel}
            show={props.visible}
        >
            <div>
                {'Are you sure you want to release this legal hold? All data associated with it will immediately be deleted and cannot be recovered.'}
            </div>
        </GenericModal>
    );
};

export default ConfirmRelease;

