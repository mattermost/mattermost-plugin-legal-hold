import React from 'react';

type Props = React.PropsWithChildren<{
    id: string;
    name: string;
    helpText: string;
}>;

const BaseSetting = (props: Props) => {
    return (
        <div
            id={`legal-hold-admin-console-setting-${props.id}`}
            className='form-group'
        >
            <label
                data-testid={`${props.id}-label`}
                htmlFor={props.id}
                className='control-label col-sm-4'
            >
                {props.name && `${props.name}:`}
            </label>
            <div className='col-sm-8'>
                {props.children}
                <div
                    className='help-text'
                >
                    <span>{props.helpText}</span>
                </div>
            </div>
        </div>
    );
};

export default BaseSetting;
