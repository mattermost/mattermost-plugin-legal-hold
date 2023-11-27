import React from 'react';

interface CreateLegalHoldButtonProps {
    onClick: () => void;
}

const CreateLegalHoldButton = (props: CreateLegalHoldButtonProps) => {
    return (
        <button
            type='submit'
            data-testid='create'
            id='createLegalHold'
            className='btn btn-primary'
            onClick={props.onClick}
        >
            {'Create new'}
        </button>
    );
};

export default CreateLegalHoldButton;
