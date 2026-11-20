import {Button} from '@mattermost/compass-ui/components/button';
import {Modal, ModalOverlay} from '@mattermost/compass-ui/components/modal';

import type {LegalHold} from '@/types';

import './legal_hold_form.css';

interface ShowSecretModalProps {
    legalHold: LegalHold | null;
    onClose: () => void;
    visible: boolean;
}

const ShowSecretModal = (props: ShowSecretModalProps) => {
    const lh = props.legalHold;

    return (
        <ModalOverlay
            open={props.visible}
            onClose={props.onClose}
        >
            <Modal
                open={props.visible}
                onClose={props.onClose}
                title='Legal hold secret'
                size='medium'
                headerDivider={false}
                footerDivider={false}
                footer={(
                    <Button
                        type='button'
                        onClick={props.onClose}
                    >
                        {'Close'}
                    </Button>
                )}
            >
                {lh && (
                    <div className='legal-hold-secret'>
                        <p>{'The secret key to check the authenticity of the legal hold is:'}</p>
                        <code className='legal-hold-secret__value'>{lh.secret}</code>
                        <p>{'Please keep this key safe and do not share it with anyone.'}</p>
                        <p>{'In order to verify the contents of the files in a legal hold, ensure you put the following flag in the processor command execution:'}</p>
                        <code className='legal-hold-secret__value'>{`--legal-hold-secret ${lh.secret}`}</code>
                    </div>
                )}
            </Modal>
        </ModalOverlay>
    );
};

export default ShowSecretModal;
