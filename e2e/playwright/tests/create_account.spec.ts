import { test, expect } from '@playwright/test';


//Go to localhost server
test ("User is able to access dev environment", async ({page}) => {
    await page.goto ('http://localhost:8065/');


    //Expect the url to contain signup user complete
    await expect (page).toHaveURL (/signup_user_complete/);


} );


// select the option to view in browser
test ('click the view in browser button', async ({page}) =>{
   await page.getByRole('link', { name: 'View in Browser' }).click();


})


//fill signup form
test ('User fills signup form', async({page}) =>{


   await page.getByPlaceholder('Email address').fill('sheila@worx$you.com');
   await page.getByPlaceholder('Choose a Username').fill('sheila');
   await page.getByPlaceholder('Choose a Password').fill('password');
   await expect(page.getByTestId('status')).toHaveText('Submitted'); 


});