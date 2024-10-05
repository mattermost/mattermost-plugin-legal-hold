import React, {useState} from 'react';

import {LegalHold} from '@/types';
import {GenericModal} from '@/components/mattermost-webapp/generic_modal/generic_modal';

import './create_legal_hold_form.scss';

interface ConfirmRunProps {
    legalHold: LegalHold | null;
    runLegalHold: (id: string) => Promise<any>;
    onExited: () => void;
    visible: boolean;
}

const ConfirmRun = (props: ConfirmRunProps) => {
    const [saving, setSaving] = useState(false);
    const [serverError, setServerError] = useState('');

    const run = () => {
        if (!props.legalHold) {
            return;
        }

        setSaving(true);
        props.runLegalHold(props.legalHold.id).then((_) => {
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

    const runMessage = (lh: LegalHold|null) => {
        if (lh) {
            return (
                <React.Fragment>
                    {'You have requested to run the following Legal Hold: '}
                    <strong>{'"'}{lh.display_name}{'"'}</strong>
                    {'This will schedule the legal hold to run as soon as possible, updating it to the current point ' +
                     'in time. In a few minutes you will be able to download the legal hold data.'}
                </React.Fragment>
            );
        }

        return (
            <React.Fragment/>
        );
    };

    const message = runMessage(props.legalHold);

    return (
        <GenericModal
            id='confirm-run-legal-hold-modal'
            className='confirm-run-legal-hold-modal'
            modalHeaderText='Run legal hold'
            confirmButtonText='Run now'
            cancelButtonText='Cancel'
            errorText={serverError}
            autoCloseOnConfirmButton={false}
            compassDesign={true}
            isConfirmDisabled={saving}
            handleConfirm={run}
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

export default ConfirmRun;

