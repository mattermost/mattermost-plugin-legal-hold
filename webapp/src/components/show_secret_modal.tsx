import React from 'react';

import {LegalHold} from '@/types';
import {GenericModal} from '@/components/mattermost-webapp/generic_modal/generic_modal';

import './create_legal_hold_form.scss';

interface ShowSecretModalProps {
    legalHold: LegalHold | null;
    onExited: () => void;
    visible: boolean;
}

const ShowSecretModal = (props: ShowSecretModalProps) => {
    const onCancel = () => {
        props.onExited();
    };

    const createMessage = (lh: LegalHold|null) => {
        if (lh) {
            return (
                <React.Fragment style={{textAlign: 'center'}}>
                    <div
                        style={{
                            textAlign: 'center',
                        }}
                    >
                        <p>{'The secret key to check the authenticity of the legal hold is: '}</p>
                        <p>
                            <code
                                style={{
                                    fontSize: '1.5em',
                                    fontWeight: 'bold',
                                    padding: '10px',
                                }}
                            >{lh.secret}</code>
                        </p>
                        <p>{'Please keep this key safe and do not share it with anyone.'}</p>
                        <hr />
                        <p>
                            {'In order to verify the contents of the files in a legal hold, ensure you put the '}
                        </p>
                        <p><code>{'--legal-hold-secret'} {lh.secret}</code></p>
                        <p>
                            {' flag in the processor command execution.'}
                        </p>
                    </div>
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
            id='show-secret-legal-hold-modal'
            className='show-secret-legal-hold-modal'
            modalHeaderText={'Legal hold secret'}
            cancelButtonText={'Close'}
            compassDesign={true}
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

export default ShowSecretModal;
