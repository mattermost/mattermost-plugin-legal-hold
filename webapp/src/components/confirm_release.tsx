import React, {useState} from 'react';

import {UserProfile} from 'mattermost-redux/types/users';

import UsersInput from '@/components/users_input';
import {CreateLegalHold, LegalHold} from '@/types';
import {GenericModal} from '@/components/mattermost-webapp/generic_modal/generic_modal';
import Input from '@/components/mattermost-webapp/input/input';

import './create_legal_hold_form.scss';

interface ConfirmReleaseProps {
    legalHold: LegalHold | null;
    releaseLegalHold: (id: string) => Promise<any>;
    onExited: () => void;
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
            props.onExited();
            setSaving(false);
        }).catch((error) => {
            setSaving(false);
            setServerError(error.toString());
        });
    };

    const onCancel = () => {
        props.onExited();
    };

    const createMessage = (lh: LegalHold|null) => {
        if (lh) {
            return (
                <React.Fragment>
                    {'Are you sure you want to release the legal hold '}
                    <strong>{'"'}{lh.display_name}{'"'}</strong>
                    {'? All data associated with it will immediately be deleted and cannot be recovered.'}
                </React.Fragment>
            );
        }

        return (
            <React.Fragment/>
        );
    };

    const message = createMessage(props.legalHold);

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
            isDeleteModal={true}
        >
            <div>
                {message}
            </div>
        </GenericModal>
    );
};

export default ConfirmRelease;

